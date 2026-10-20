"""Offline release tests: no real GitHub commands, tags, releases or installs."""
import copy
import io
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import zipfile

import release_policy as policy

SHA = "a" * 40
ENV = {
    "GITHUB_ACTIONS": "true", "GITHUB_EVENT_NAME": "workflow_dispatch",
    "GITHUB_SERVER_URL": "https://github.com", "GITHUB_REPOSITORY": policy.REPOSITORY,
    "DEFAULT_BRANCH": "main", "GITHUB_REF": "refs/heads/main", "GITHUB_SHA": SHA,
    "GITHUB_WORKFLOW_REF": policy.REPOSITORY + "/.github/workflows/release.yml@refs/heads/main",
    "SOURCE_SHA": SHA, "VERSION_TAG": "v1.2.3",
}
RESULTS = {key: "success" for key in ("prepare", "quality", "build")}


class FakeGitHub:
    def __init__(self):
        self.mutations = False
        self.calls = []
        self.tag = None
        self.release = None
        self.uploads = {}
        self.bad_digest = False
        self.missing_digest = False
        self.change_tag_after_draft = False

    def call(self, method, path, data=None, absent=False):
        self.calls.append((method, path, data))
        if method != "GET":
            policy.require(self.mutations, "Mutation was not enabled")
        if method == "GET" and path == policy.endpoint(""):
            return {"default_branch": "main"}
        if method == "GET" and path == policy.endpoint("/commits/" + SHA):
            return {"sha": SHA}
        if method == "GET" and path.startswith(policy.endpoint("/git/ref/tags/")):
            return self.tag
        if method == "GET" and path.startswith(policy.endpoint("/releases/tags/")):
            return self.release
        if method == "POST" and path == policy.endpoint("/git/refs"):
            policy.require(self.tag is None, "Ref already exists")
            self.tag = {"ref": data["ref"], "object": {"type": "commit", "sha": data["sha"]}}
            return self.tag
        if method == "PATCH" and path == policy.endpoint("/releases/123"):
            self.release.update(data)
            return self.release
        raise AssertionError("Unexpected fake API operation: " + method + " " + path)

    def create_draft(self, ctx, files):
        policy.require(self.mutations, "Mutation was not enabled")
        self.calls.append(("DRAFT", ctx["tag"], None))
        assets = []
        for number, path in enumerate(files, 1):
            content = path.read_bytes()
            self.uploads[number] = content
            digest = None if self.missing_digest else "sha256:" + policy.digest(content)
            if self.bad_digest:
                digest = "sha256:" + "0" * 64
            assets.append({"id": number, "name": path.name, "size": len(content), "digest": digest})
        self.release = {"id": 123, "draft": True, "tag_name": ctx["tag"], "target_commitish": ctx["source_sha"], "assets": assets, "html_url": "https://github.com/" + policy.REPOSITORY + "/releases/tag/" + ctx["tag"]}
        if self.change_tag_after_draft:
            self.tag["object"]["sha"] = "b" * 40

    def download_asset(self, asset_id):
        return self.uploads[asset_id]


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.live_commands = patch.object(subprocess, "run", side_effect=AssertionError("Live subprocess is forbidden in offline tests"))
        self.live_commands.start()
        self.addCleanup(self.live_commands.stop)
        self.temporary = tempfile.TemporaryDirectory(prefix="release-policy-test-")
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.ctx = policy.context(ENV)

    def packages(self):
        for platform, (package, extension, binary, goos, _) in policy.PLATFORMS.items():
            target = self.directory / (package + "-" + self.ctx["version"] + "." + extension)
            data = ("synthetic binary " + platform).encode()
            if extension == "zip":
                with zipfile.ZipFile(target, "w") as archive:
                    archive.writestr(binary, data)
            else:
                with tarfile.open(target, "w:gz") as archive:
                    info = tarfile.TarInfo(binary)
                    info.size = len(data)
                    info.mode = 0o755
                    archive.addfile(info, io.BytesIO(data))
            policy.manifest(self.ctx, platform, self.directory, goos, "arm64" if platform == "macos" else "amd64", "synthetic Go version")

    def test_valid_semantic_versions(self):
        for value in ("v0.0.0", "v1.2.3", "v1.2.3-rc.1", "v1.2.3+build.001", "v1.2.3-beta.0+metadata"):
            with self.subTest(value=value):
                self.assertEqual(policy.version(value)[0], value[1:])

    def test_invalid_and_injection_versions(self):
        for value in (None, "", "1.2.3", "v01.2.3", "v1.02.3", "v1.2.03", "v1.2", "v1.2.3-01", "v1.2.3-rc.01", "v1.2.3/other", "v1.2.3\n", "v1.2.3;echo", "v1.2.3$(id)", "v1.2.3_foo", "v1.2.3+", "v1.2.3-", "v" + "9" * 81, "v1000000000.1.1", "v1.2.3-☃"):
            with self.subTest(value=value), self.assertRaises(policy.ReleaseError):
                policy.version(value)

    def test_nonmanual_events_are_rejected(self):
        for event in ("push", "pull_request", "release", "workflow_run", "workflow_call", "repository_dispatch", "schedule"):
            with self.subTest(event=event), self.assertRaises(policy.ReleaseError):
                policy.context({**ENV, "GITHUB_EVENT_NAME": event})

    def test_context_rejects_local_wrong_ref_repo_and_source(self):
        cases = [{"GITHUB_ACTIONS": "false"}, {"GITHUB_REF": "refs/tags/v1.2.3"}, {"GITHUB_REF": "refs/heads/feature"}, {"GITHUB_REPOSITORY": "other/repo"}, {"GITHUB_SERVER_URL": "https://other.invalid"}, {"GITHUB_WORKFLOW_REF": policy.REPOSITORY + "/.github/workflows/ci.yml@refs/heads/main"}, {"SOURCE_SHA": "b" * 40}, {"SOURCE_SHA": "A" * 40}, {"SOURCE_SHA": "main"}, {"SOURCE_SHA": SHA + "\n"}]
        for change in cases:
            with self.subTest(change=change), self.assertRaises(policy.ReleaseError):
                policy.context({**ENV, **change})

    def test_preflight_is_read_only(self):
        api = FakeGitHub()
        policy.preflight(self.ctx, api)
        self.assertFalse(api.mutations)
        self.assertTrue(all(method == "GET" for method, _, _ in api.calls))

    def test_existing_tag_is_rejected_without_writes(self):
        api = FakeGitHub()
        api.tag = {"object": {"sha": SHA}}
        with self.assertRaises(policy.ReleaseError):
            policy.preflight(self.ctx, api)
        self.assertFalse(api.mutations)
        self.assertTrue(all(method == "GET" for method, _, _ in api.calls))

    def test_existing_release_or_draft_is_rejected(self):
        for draft in (True, False):
            api = FakeGitHub()
            api.release = {"draft": draft}
            with self.subTest(draft=draft), self.assertRaises(policy.ReleaseError):
                policy.preflight(self.ctx, api)
            self.assertFalse(api.mutations)

    def test_http_404_is_the_only_absence(self):
        for status in (401, 403, 429, 500):
            runner = lambda *args, **kwargs: SimpleNamespace(returncode=1, stdout="HTTP/2.0 " + str(status) + " Error\r\n\r\n{}")
            with self.subTest(status=status), self.assertRaises(policy.ReleaseError):
                policy.GitHub(runner).call("GET", "repos/example", absent=True)
        missing = lambda *args, **kwargs: SimpleNamespace(returncode=1, stdout="HTTP/2.0 404 Not Found\r\n\r\n{}")
        self.assertIsNone(policy.GitHub(missing).call("GET", "repos/example", absent=True))

    def test_http_headers_and_json_are_parsed(self):
        self.assertEqual(policy.http_response('HTTP/2.0 200 OK\r\nContent-Type: application/json\r\n\r\n{"value":1}'), (200, {"value": 1}))
        with self.assertRaises(policy.ReleaseError):
            policy.http_response('{"value":1}')

    def test_default_client_cannot_mutate(self):
        with self.assertRaises(policy.ReleaseError):
            policy.GitHub().call("POST", "repos/example/git/refs", {})

    def test_all_three_package_formats_and_provenance(self):
        self.packages()
        files = policy.verify_packages(self.ctx, self.directory)
        self.assertEqual(len(files), 5)
        self.assertEqual(len((self.directory / "SHA256SUMS").read_text().splitlines()), 3)
        metadata = json.loads((self.directory / "release-provenance.json").read_text())
        self.assertEqual(metadata["source_sha"], SHA)
        self.assertEqual({p["platform"] for p in metadata["packages"]}, set(policy.PLATFORMS))

    def test_checksum_corruption_prevents_any_api_call(self):
        self.packages()
        path = self.directory / "asteroids-linux-amd64.manifest.json"
        data = json.loads(path.read_text())
        data["sha256"] = "0" * 64
        path.write_text(json.dumps(data))
        api = FakeGitHub()
        with self.assertRaises(policy.ReleaseError):
            policy.publish(self.ctx, self.directory, api, RESULTS)
        self.assertEqual(api.calls, [])
        self.assertFalse(api.mutations)

    def test_wrong_source_manifest_is_rejected(self):
        self.packages()
        path = self.directory / "asteroids-macos.manifest.json"
        data = json.loads(path.read_text())
        data["source_sha"] = "b" * 40
        path.write_text(json.dumps(data))
        with self.assertRaises(policy.ReleaseError):
            policy.verify_packages(self.ctx, self.directory)

    def test_extra_artifact_is_rejected(self):
        self.packages()
        (self.directory / "unexpected").write_text("synthetic")
        with self.assertRaises(policy.ReleaseError):
            policy.verify_packages(self.ctx, self.directory)

    def test_wrong_archive_member_is_rejected(self):
        path = self.directory / "bad.zip"
        with zipfile.ZipFile(path, "w") as archive:
            archive.writestr("../escape.exe", b"synthetic")
        with self.assertRaises(policy.ReleaseError):
            policy.archive_bytes(path, "asteroids.exe")

    def test_failed_skipped_or_cancelled_checks_prevent_all_calls(self):
        for key in RESULTS:
            for result in ("failure", "cancelled", "skipped", ""):
                api = FakeGitHub()
                with self.subTest(key=key, result=result), self.assertRaises(policy.ReleaseError):
                    policy.publish(self.ctx, self.directory, api, {**RESULTS, key: result})
                self.assertEqual(api.calls, [])

    def test_new_tag_then_verified_draft_then_publish(self):
        self.packages()
        api = FakeGitHub()
        result = policy.publish(self.ctx, self.directory, api, RESULTS)
        self.assertFalse(result["draft"])
        mutations = [(method, path, data) for method, path, data in api.calls if method != "GET"]
        self.assertEqual([m[0] for m in mutations], ["POST", "DRAFT", "PATCH"])
        self.assertEqual(mutations[0][2], {"ref": "refs/tags/v1.2.3", "sha": SHA})
        self.assertEqual(mutations[-1][2], {"draft": False})

    def test_bad_uploaded_digest_leaves_draft_without_publication(self):
        self.packages()
        api = FakeGitHub()
        api.bad_digest = True
        with self.assertRaises(policy.ReleaseError):
            policy.publish(self.ctx, self.directory, api, RESULTS)
        self.assertTrue(api.release["draft"])
        self.assertFalse(any(method == "PATCH" for method, _, _ in api.calls))

    def test_missing_server_digest_is_verified_from_downloaded_bytes(self):
        self.packages()
        api = FakeGitHub()
        api.missing_digest = True
        self.assertFalse(policy.publish(self.ctx, self.directory, api, RESULTS)["draft"])

    def test_tag_drift_prevents_draft_publication(self):
        self.packages()
        api = FakeGitHub()
        api.change_tag_after_draft = True
        with self.assertRaises(policy.ReleaseError):
            policy.publish(self.ctx, self.directory, api, RESULTS)
        self.assertTrue(api.release["draft"])
        self.assertFalse(any(method == "PATCH" for method, _, _ in api.calls))

    def test_toolchain_selection_uses_existing_native_cache(self):
        root = self.directory / "go"
        name = "go.exe" if policy.os.name == "nt" else "go"
        for number in ("1.24.10", "1.26.9"):
            binary = root / number / "x64/bin" / name
            binary.parent.mkdir(parents=True)
            binary.write_bytes(b"synthetic, never executed")
        self.assertEqual(policy.select_go(root, "X64", (1, 24, 10)).parents[2].name, "1.26.9")
        with self.assertRaises(policy.ReleaseError):
            policy.select_go(root, "ARM64", (1, 24, 10))
        with self.assertRaises(policy.ReleaseError):
            policy.select_go(root, "X64", (1, 99, 0))


if __name__ == "__main__":
    unittest.main()
