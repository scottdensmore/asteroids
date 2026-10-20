"""Manual-release guards. Importing this module performs no network or writes.

Publication is confined to the explicit publish subcommand in a manual Actions
run from the default branch. Tests use fake GitHub clients, never real tags.
"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tarfile
import urllib.parse
import zipfile

REPOSITORY = "scottdensmore/asteroids"
MAX_ASSET = 512 * 1024 * 1024
PLATFORMS = {
    "linux-amd64": ("asteroids-linux-amd64", "tar.gz", "asteroids", "linux", {"amd64"}),
    "windows-amd64": ("asteroids-windows-amd64", "zip", "asteroids.exe", "windows", {"amd64"}),
    "macos": ("asteroids-macos", "tar.gz", "asteroids", "darwin", {"amd64", "arm64"}),
}
TAG = re.compile(r"v(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?\Z")


class ReleaseError(ValueError):
    pass


def require(condition, message):
    if not condition:
        raise ReleaseError(message)


def version(tag):
    require(isinstance(tag, str) and len(tag) <= 80, "Version tag must be at most 80 characters")
    match = TAG.fullmatch(tag)
    require(match is not None, "Expected a bounded vMAJOR.MINOR.PATCH semantic version")
    prerelease = match.group(4)
    if prerelease:
        require(all(not (p.isdigit() and len(p) > 1 and p.startswith("0")) for p in prerelease.split(".")), "Numeric prerelease identifiers cannot have leading zeroes")
    return tag[1:], bool(prerelease)


def context(env):
    require(env.get("GITHUB_ACTIONS") == "true", "Publication requires GitHub Actions")
    require(env.get("GITHUB_EVENT_NAME") == "workflow_dispatch", "Only workflow_dispatch is allowed")
    require(env.get("GITHUB_SERVER_URL") == "https://github.com", "Unexpected GitHub server")
    require(env.get("GITHUB_REPOSITORY") == REPOSITORY, "Unexpected repository")
    branch = env.get("DEFAULT_BRANCH", "")
    require(branch and env.get("GITHUB_REF") == "refs/heads/" + branch, "Dispatch must use the default branch")
    require(env.get("GITHUB_WORKFLOW_REF") == REPOSITORY + "/.github/workflows/release.yml@refs/heads/" + branch, "Unexpected workflow source")
    source = env.get("SOURCE_SHA", "")
    require(re.fullmatch(r"[0-9a-f]{40}", source) is not None, "Source must be a full lowercase commit SHA")
    require(source == env.get("GITHUB_SHA"), "Source must equal the immutable default-branch dispatch commit")
    tag = env.get("VERSION_TAG", "")
    normalized, prerelease = version(tag)
    return {"tag": tag, "version": normalized, "prerelease": prerelease, "source_sha": source, "default_branch": branch}


def http_response(raw):
    raw = raw.replace("\r\n", "\n")
    while True:
        headers, separator, body = raw.partition("\n\n")
        require(bool(separator), "GitHub response omitted HTTP headers")
        match = re.match(r"HTTP/\S+\s+([0-9]{3})", headers)
        require(match is not None, "Invalid GitHub HTTP response")
        if body.startswith("HTTP/"):
            raw = body
            continue
        return int(match.group(1)), json.loads(body) if body.strip() else None


class GitHub:
    def __init__(self, runner=None):
        self.runner = runner or subprocess.run
        self.mutations = False

    def call(self, method, path, data=None, absent=False):
        require(method == "GET" or self.mutations, "Mutation is not enabled")
        args = ["/usr/bin/gh", "api", "--include", "--method", method, path]
        if data is not None:
            args += ["--input", "-"]
        result = self.runner(args, input=json.dumps(data) if data is not None else None, text=True, capture_output=True, check=False)
        status, body = http_response(result.stdout)
        if absent and status == 404:
            return None
        require(result.returncode == 0 and status in (200, 201), "GitHub API failed closed: HTTP " + str(status))
        return body

    def create_draft(self, ctx, files):
        require(self.mutations, "Mutation is not enabled")
        args = ["/usr/bin/gh", "release", "create", ctx["tag"], *map(str, files), "--repo", REPOSITORY, "--verify-tag", "--target", ctx["source_sha"], "--title", ctx["tag"], "--generate-notes", "--draft"]
        if ctx["prerelease"]:
            args.append("--prerelease")
        result = self.runner(args, text=True, capture_output=True, check=False)
        require(result.returncode == 0, "Draft creation failed; inspect the tag/draft manually, do not clobber or resume")

    def download_asset(self, asset_id):
        require(type(asset_id) is int and asset_id > 0, "Invalid uploaded asset ID")
        result = self.runner(["/usr/bin/gh", "api", "--method", "GET", "-H", "Accept: application/octet-stream", "repos/" + REPOSITORY + "/releases/assets/" + str(asset_id)], capture_output=True, check=False)
        require(result.returncode == 0 and len(result.stdout) <= MAX_ASSET, "Cannot verify uploaded asset bytes")
        return result.stdout


def endpoint(suffix):
    return "repos/" + REPOSITORY + suffix


def tag_path(ctx):
    return endpoint("/git/ref/tags/" + urllib.parse.quote(ctx["tag"], safe=""))


def release_path(ctx):
    return endpoint("/releases/tags/" + urllib.parse.quote(ctx["tag"], safe=""))


def preflight(ctx, api):
    repo = api.call("GET", endpoint(""))
    require(repo.get("default_branch") == ctx["default_branch"], "Default branch changed; review before retrying")
    commit = api.call("GET", endpoint("/commits/" + ctx["source_sha"]))
    require(commit.get("sha") == ctx["source_sha"], "Source commit did not resolve exactly")
    require(api.call("GET", tag_path(ctx), absent=True) is None, "Tag already exists; never replace it")
    require(api.call("GET", release_path(ctx), absent=True) is None, "Release already exists, including drafts; never resume or overwrite it")


def archive_bytes(path, binary):
    require(path.is_file() and not path.is_symlink() and 0 < path.stat().st_size <= MAX_ASSET, "Invalid archive file")
    if path.name.endswith(".zip"):
        with zipfile.ZipFile(path) as archive:
            members = archive.infolist()
            require(len(members) == 1 and members[0].filename == binary and not members[0].is_dir(), "Unexpected ZIP members")
            require(not stat.S_ISLNK(members[0].external_attr >> 16) and not members[0].flag_bits & 1 and members[0].file_size <= MAX_ASSET, "Unsafe ZIP member")
            return archive.read(members[0])
    with tarfile.open(path, "r:gz") as archive:
        members = archive.getmembers()
        require(len(members) == 1 and members[0].name == binary and members[0].isfile() and members[0].size <= MAX_ASSET, "Unexpected tar members")
        return archive.extractfile(members[0]).read()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def manifest(ctx, platform, directory, goos, goarch, go_version):
    require(platform in PLATFORMS, "Unknown package platform")
    package, extension, binary, expected_os, arches = PLATFORMS[platform]
    require(goos == expected_os and goarch in arches, "Package architecture does not match its platform")
    path = directory / (package + "-" + ctx["version"] + "." + extension)
    content = archive_bytes(path, binary)
    data = {"schema": 1, "tag": ctx["tag"], "version": ctx["version"], "source_sha": ctx["source_sha"], "platform": platform, "goos": goos, "goarch": goarch, "go_version": go_version, "archive": path.name, "size": path.stat().st_size, "sha256": digest(path.read_bytes()), "binary_sha256": digest(content)}
    (directory / (package + ".manifest.json")).write_text(json.dumps(data, indent=2) + "\n")
    return data


def verify_packages(ctx, directory):
    require(directory.is_dir() and not directory.is_symlink(), "Missing package directory")
    expected = set()
    manifests = []
    files = []
    for platform, (package, extension, binary, goos, arches) in PLATFORMS.items():
        name = package + ".manifest.json"
        path = directory / name
        require(path.is_file() and not path.is_symlink() and path.stat().st_size < 65536, "Invalid manifest")
        data = json.loads(path.read_text())
        archive_name = package + "-" + ctx["version"] + "." + extension
        require(data.get("schema") == 1 and data.get("platform") == platform and data.get("goos") == goos and data.get("goarch") in arches, "Manifest platform mismatch")
        require(all(data.get(k) == ctx[k] for k in ("tag", "version", "source_sha")), "Manifest source/version mismatch")
        require(data.get("archive") == archive_name, "Manifest archive mismatch")
        archive = directory / archive_name
        content = archive_bytes(archive, binary)
        require(data.get("size") == archive.stat().st_size and data.get("sha256") == digest(archive.read_bytes()) and data.get("binary_sha256") == digest(content), "Package checksum mismatch")
        expected.update((name, archive_name))
        manifests.append(data)
        files.append(archive)
    require({p.name for p in directory.iterdir()} == expected, "Unexpected or missing artifact files")
    checksums = directory / "SHA256SUMS"
    checksums.write_text("".join(d["sha256"] + "  " + d["archive"] + "\n" for d in manifests))
    provenance = directory / "release-provenance.json"
    provenance.write_text(json.dumps({"tag": ctx["tag"], "source_sha": ctx["source_sha"], "packages": manifests}, indent=2) + "\n")
    return files + [checksums, provenance]


def publish(ctx, directory, api, results):
    require(all(results.get(k) == "success" for k in ("prepare", "quality", "build")), "All build/test jobs must succeed before publication")
    files = verify_packages(ctx, directory)
    preflight(ctx, api)  # Recheck immediately before the first mutation.
    api.mutations = True
    ref = api.call("POST", endpoint("/git/refs"), {"ref": "refs/tags/" + ctx["tag"], "sha": ctx["source_sha"]})
    require(ref.get("ref") == "refs/tags/" + ctx["tag"] and ref.get("object", {}).get("sha") == ctx["source_sha"] and ref["object"].get("type") == "commit", "Created tag did not target the tested commit")
    api.create_draft(ctx, files)
    release = api.call("GET", release_path(ctx))
    require(release.get("tag_name") == ctx["tag"] and release.get("target_commitish") == ctx["source_sha"] and release.get("draft") is True, "Draft provenance mismatch")
    expected = {f.name: f for f in files}
    assets = release.get("assets", [])
    require(len(assets) == len(expected) and {a.get("name") for a in assets} == set(expected), "Uploaded asset set mismatch")
    for asset in assets:
        path = expected[asset["name"]]
        require(asset.get("size") == path.stat().st_size, "Uploaded asset size mismatch")
        actual_digest = asset.get("digest")
        if actual_digest is None:
            actual_digest = "sha256:" + digest(api.download_asset(asset.get("id")))
        require(actual_digest == "sha256:" + digest(path.read_bytes()), "Uploaded asset checksum mismatch")
    ref = api.call("GET", tag_path(ctx))
    require(ref.get("object", {}).get("sha") == ctx["source_sha"] and ref["object"].get("type") == "commit", "Tag changed; draft must not be published")
    require(type(release.get("id")) is int and release["id"] > 0, "Invalid draft ID")
    result = api.call("PATCH", endpoint("/releases/" + str(release["id"])), {"draft": False})
    require(result.get("draft") is False and result.get("tag_name") == ctx["tag"], "Release publication was not confirmed")
    return result


def select_go(root, arch, minimum):
    require(arch in ("X64", "ARM64"), "Unsupported runner architecture")
    folder = "x64" if arch == "X64" else "arm64"
    executable = "go.exe" if os.name == "nt" else "go"
    choices = []
    for binary in root.glob("*/" + folder + "/bin/" + executable):
        match = re.fullmatch(r"([0-9]+)\.([0-9]+)\.([0-9]+)", binary.parents[2].name)
        if match:
            number = tuple(map(int, match.groups()))
            if number >= minimum:
                choices.append((number, binary))
    require(bool(choices), "No compatible preinstalled native Go; installation is not authorized")
    return max(choices)[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=("prepare", "manifest", "publish", "toolchain"))
    parser.add_argument("--platform", choices=tuple(PLATFORMS))
    parser.add_argument("--directory", type=Path, default=Path("package"))
    args = parser.parse_args()
    if args.mode == "toolchain":
        matches = re.findall(r"^(?:go |toolchain go)([0-9]+)\.([0-9]+)(?:\.([0-9]+))?$", Path("go.mod").read_text(), re.M)
        require(bool(matches), "Go minimum version is missing")
        minimum = max(tuple(int(v or 0) for v in m) for m in matches)
        binary = select_go(Path(os.environ["RUNNER_TOOL_CACHE"]) / "go", os.environ["RUNNER_ARCH"], minimum)
        subprocess.run([str(binary), "version"], check=True)
        with open(os.environ["GITHUB_PATH"], "a") as output:
            output.write(str(binary.parent) + "\n")
        return
    ctx = context(os.environ)
    if args.mode == "prepare":
        preflight(ctx, GitHub())
        ctx["build_date"] = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        with open(os.environ["GITHUB_OUTPUT"], "a") as output:
            for key in ("tag", "version", "source_sha", "build_date"):
                output.write(key + "=" + ctx[key] + "\n")
    elif args.mode == "manifest":
        require(subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip() == ctx["source_sha"], "Build checkout drifted from the selected source")
        goos, goarch = subprocess.check_output(["go", "env", "GOOS", "GOARCH"], text=True).splitlines()
        go_version = subprocess.check_output(["go", "version"], text=True).strip()
        manifest(ctx, args.platform, args.directory, goos, goarch, go_version)
    else:
        result = publish(ctx, args.directory, GitHub(), {k: os.environ.get("RELEASE_" + k.upper() + "_RESULT") for k in ("prepare", "quality", "build")})
        print("Created release", result.get("html_url", ctx["tag"]), "for", ctx["source_sha"])


if __name__ == "__main__":
    main()
