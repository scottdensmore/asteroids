# **Asteroids Clone**

## **1\. Project Overview**

The objective is to create a clone of the classic 1979 arcade game "Asteroids." The game must be written in idiomatic Go and must replicate the look, feel, and core mechanics of the original. This includes vector-style graphics, ship inertia, screen-wrapping, and splitting asteroids.

## **2\. Core Technologies**

* **Language:** Go (Golang)  
* **Graphics Library:** [Ebitengine (formerly Ebiten)](https://ebitengine.org/)  
  * Reasoning: Ebitengine is a simple,  
    well-documented, and high-performance 2D game library for Go. It provides a clean API that handles the game loop, input (keyboard/gamepad), and rendering pipeline, allowing us to focus on game logic.

## **3\. Design & Aesthetics (The "Classic" Look)**

The game **must** emulate the classic monochrome vector graphics of the original.

* **Rendering:** All game objects (ship, asteroids, bullets) will be rendered as geometric outlines (polygons) drawn with white lines. **Do not use sprites or filled shapes.**  
* **Tooling:** Use ebitenutil.DrawLine or vector.StrokeLine for all drawing. Ebitengine is a raster library, but we will use it to simulate vector graphics.  
* **Background:** The background must be solid black.  
* **UI:** Text for score and lives should be rendered in a simple, pixelated, or fixed-width font (e.g., golang.org/x/image/font/basicfont).

## **4\. Core Game Entities & Data Structures**

You will need to define structs to represent the game objects. All positions and velocities should use float64.  
// A simple 2D vector for position and velocity  
type Vector2D struct {  
    X, Y float64  
}

// Player's ship  
type Ship struct {  
    Position      Vector2D  
    Velocity      Vector2D  
    Rotation      float64 // In radians  
    IsThrusting   bool  
    IsInvincible  bool  
    InvincibleTimer float64  
}

// An asteroid  
type Asteroid struct {  
    Position      Vector2D  
    Velocity      Vector2D  
    Rotation      float64 // In radians  
    RotationSpeed float64  
    Shape         \[\]Vector2D // Vertices relative to position (e.g., a polygon)  
    Size          int        // e.g., 3 \= Large, 2 \= Medium, 1 \= Small  
    Radius        float64    // For simple collision detection  
}

// A bullet fired by the ship  
type Bullet struct {  
    Position Vector2D  
    Velocity Vector2D  
    Lifespan float64 // Time in seconds before it disappears  
}

// Main game state struct  
type Game struct {  
    Ship        \*Ship  
    Asteroids   \[\]\*Asteroid  
    Bullets     \[\]\*Bullet  
    Score       int  
    Lives       int  
    Level       int  
    ScreenWidth int  
    ScreenHeight int  
    GameState   int // e.g., 0=Playing, 1=GameOver, 2=TitleScreen  
    LastShot    time.Time  
}

## **5\. Step-by-Step Implementation Plan**

Follow this plan sequentially. Each step builds on the last and should result in a testable, runnable state.

### **Step 1: Project Setup & Blank Window**

1. **Action:**  
   * Initialize a new Go module (go mod init).  
   * Add Ebitengine as a dependency (go get github.com/hajimehoshi/ebiten/v2).  
   * Create a main.go file.  
   * Define the Game struct and implement the ebiten.Game interface (Update, Draw, Layout).  
   * In main(), set the window size (e.g., 800x600), title, and run the game loop with ebiten.RunGame(NewGame()).  
2. **Test:** Running go run . should open a blank, black 800x600 window.

### **Step 2: Draw the Ship (Static)**

1. **Action:**  
   * Initialize a Ship object within the Game struct.  
   * Define the ship's shape as a simple triangle (3 Vector2D points).  
   * In the Game.Draw method, draw the ship's outline (three lines) in the center of the screen.  
2. **Test:** The window should display a static, white, triangular ship in the center.

### **Step 3: Ship Rotation**

1. **Action:**  
   * In Game.Update, check for input from the left and right arrow keys (ebiten.KeyLeft, ebiten.KeyRight).  
   * If a key is pressed, add or subtract from Ship.Rotation (e.g., Ship.Rotation \+= 0.05).  
   * In Game.Draw, use ebiten.DrawImageOptions with GeoM.Rotate to rotate the ship's drawing around its center based on Ship.Rotation.  
2. **Test:** The ship should rotate left and right when the arrow keys are pressed.

### **Step 4: Ship Thrust & Inertia (Core Mechanic)**

1. **Action:**  
   * In Game.Update, check for the 'Up' arrow key (ebiten.KeyUp).  
   * If pressed:  
     * Set Ship.IsThrusting \= true.  
     * Calculate acceleration based on Ship.Rotation:  
       * accelX := math.Cos(Ship.Rotation) \* THRUST\_FORCE  
       * accelY := math.Sin(Ship.Rotation) \* THRUST\_FORCE  
     * Add this acceleration to Ship.Velocity: Ship.Velocity.X \+= accelX, Ship.Velocity.Y \+= accelY.  
   * In *every* Update frame (outside the key check):  
     * Update the ship's position: Ship.Position.X \+= Ship.Velocity.X, Ship.Position.Y \+= Ship.Velocity.Y.  
   * (Optional: Add a "flame" triangle to the back of the ship in Game.Draw only when Ship.IsThrusting is true).  
2. **Test:** The ship should now move. Pressing 'Up' accelerates it. When 'Up' is released, the ship continues to drift (inertia). It should feel "slippery" and true to the original.

### **Step 5: Screen Wrapping**

1. **Action:**  
   * Create a helper function (g \*Game) wrap(pos Vector2D) Vector2D.  
   * This function checks pos.X and pos.Y against the ScreenWidth and ScreenHeight.  
   * If pos.X \< 0, set pos.X \= ScreenWidth.  
   * If pos.X \> ScreenWidth, set pos.X \= 0\.  
   * Do the same for pos.Y.  
   * In Game.Update, after updating Ship.Position, call Ship.Position \= g.wrap(Ship.Position).  
2. **Test:** The ship should disappear from one edge of the screen and reappear on the opposite edge.

### **Step 6: Spawning Bullets**

1. **Action:**  
   * Define the Bullet struct (see section 4).  
   * In Game.Update, check for the 'Spacebar' key (ebiten.KeySpace).  
   * Implement a firing cooldown (e.g., time.Since(g.LastShot) \> 200\*time.Millisecond).  
   * If 'Spacebar' is pressed and cooldown is over:  
     * Create a new Bullet.  
     * Set its Position to the nose of the ship.  
     * Set its Velocity based on Ship.Rotation *plus* the ship's current Velocity. This is crucial for realistic firing while moving.  
     * Set Lifespan (e.g., 1.0 second).  
     * Add the new bullet to the g.Bullets slice.  
     * Update g.LastShot \= time.Now().  
2. **Test:** Pressing 'Spacebar' should not do anything visible *yet*, but the g.Bullets slice should grow.

### **Step 7: Bullets Update & Draw**

1. **Action:**  
   * In Game.Update, iterate through the g.Bullets slice (use a reverse loop or copy-on-remove to safely delete).  
     * Update bullet.Position based on bullet.Velocity.  
     * Apply g.wrap() to bullet.Position.  
     * Decrement bullet.Lifespan by the delta time (e.g., 1.0 / float64(ebiten.TPS())).  
     * If bullet.Lifespan \<= 0, remove it from the slice.  
   * In Game.Draw, iterate through g.Bullets and draw each one (a simple white dot or 1x1 line is fine).  
2. **Test:** Pressing 'Spacebar' now fires visible bullets that travel in a straight line, wrap around the screen, and disappear after a second.

### **Step 8: Spawning Asteroids**

1. **Action:**  
   * Define the Asteroid struct (see section 4).  
   * Create a function (g \*Game) spawnAsteroids(count int, size int):  
     * It should create count new Asteroid objects.  
     * Each should have a random Position, Velocity, RotationSpeed.  
     * **Crucially:** Ensure they spawn at the edges of the screen, *not* in the center where the player is.  
     * Each needs a Shape (a list of vertices for a jagged polygon) and a Radius for collision.  
   * In your Game constructor, call g.spawnAsteroids(4, 3\) to start Level 1\.  
2. **Test:** No visual change *yet*, but the g.Asteroids slice should be populated.

### **Step 9: Asteroids Update & Draw**

1. **Action:**  
   * In Game.Update, iterate through g.Asteroids:  
     * Update asteroid.Position based on asteroid.Velocity.  
     * Update asteroid.Rotation based on asteroid.RotationSpeed.  
     * Apply g.wrap() to asteroid.Position.  
   * In Game.Draw, iterate through g.Asteroids and draw their polygon shapes using their Position, Rotation, and Shape vertices.  
2. **Test:** Four large asteroids should be visible, moving in random directions, rotating, and wrapping around the screen.

### **Step 10: Collision Detection (Bullet vs. Asteroid)**

1. **Action:**  
   * In Game.Update, create a nested loop: for i, bullet := range g.Bullets... for j, asteroid := range g.Asteroids.  
   * Implement collision logic. **Start with simple bounding circle collision:**  
     * distance := math.Hypot(bullet.Position.X \- asteroid.Position.X, bullet.Position.Y \- asteroid.Position.Y)  
     * If distance \< asteroid.Radius, a hit has occurred.  
   * If a hit occurs:  
     * Remove the bullet from g.Bullets.  
     * Call a new function, g.splitAsteroid(j).  
     * Add to g.Score.  
     * break the inner loop (bullet is gone).  
2. **Action (Splitting):**  
   * Create (g \*Game) splitAsteroid(index int).  
   * Get the asteroid := g.Asteroids\[index\].  
   * Remove it from the g.Asteroids slice.  
   * If asteroid.Size \> 1 (i.e., Large or Medium):  
     * Create two new asteroids at the same Position.  
     * Their Size should be asteroid.Size \- 1\.  
     * Give them new, diverging velocities.  
     * Add them to the g.Asteroids slice.  
   * If asteroid.Size \== 1 (Small), it is simply destroyed and does not split.  
3. **Test:** Shooting asteroids should cause them to split into smaller ones (Large \-\> 2 Medium, Medium \-\> 2 Small). Shooting small ones destroys them.

### **Step 11: Collision Detection (Ship vs. Asteroid) & Death**

1. **Action:**  
   * In Game.Update, loop for i, asteroid := range g.Asteroids.  
   * Check for collision between g.Ship and asteroid (bounding circle is fine).  
   * If a hit occurs *and* g.Ship.IsInvincible \== false:  
     * Call a new function, g.killShip().  
2. **Action (Death & Respawn):**  
   * Create (g \*Game) killShip().  
   * This function should:  
     * Decrement g.Lives.  
     * Check if g.Lives \<= 0\. If so, set g.GameState \= 1 (GameOver).  
     * If lives remain:  
       * Reset g.Ship.Position to center, Velocity to zero.  
       * Set g.Ship.IsInvincible \= true and g.Ship.InvincibleTimer \= 3.0 (3 seconds).  
   * In Game.Update, if g.Ship.IsInvincible, decrement the timer. When it hits 0, set IsInvincible \= false.  
   * In Game.Draw, if g.Ship.IsInvincible, make the ship blink (e.g., if int(g.Ship.InvincibleTimer\*10)%2 \== 0 { drawShip() }).  
3. **Test:** Crashing into an asteroid resets the ship, makes it blink, and removes a life. After 3 seconds, it can be destroyed again.

### **Step 12: UI (Score & Lives) & Game Loop**

1. **Action:**  
   * In Game.Draw, use ebitenutil.DebugPrint or text.Draw to display the g.Score and g.Lives at the top of the screen. (Drawing g.Lives as 3 small ship icons is a good polish step).  
   * In Game.Update, check if len(g.Asteroids) \== 0\. If true:  
     * Increment g.Level.  
     * Call g.spawnAsteroids(g.Level \+ 3, 3\) to start the next, harder wave.  
   * In Game.Update, if g.GameState \== 1 (GameOver), check for 'Enter' key to reset the game.  
   * In Game.Draw, if g.GameState \== 1, draw "GAME OVER" in the center.  
2. **Test:** The full game loop is now in place. You can score points, lose lives, clear levels, and get a "Game Over."

## **6\. Go Best Practices**

* **Code Organization:** Keep main.go clean. You may want to split entity logic (ship, asteroid) into separate files (e.g., ship.go, asteroid.go) in the same package.  
* **Idiomatic Go:** Use Go modules for dependencies. Format your code with gofmt or goimports.  
* **Comments:** Comment complex logic, especially physics and collision calculations.  
* **Constants:** Do not use magic numbers. Define constants for THRUST\_FORCE, BULLET\_SPEED, SHIP\_ROTATION\_SPEED, BULLET\_LIFESPAN, etc.  
* **Error Handling:** Handle errors from Ebitengine functions, though for this small project, log.Fatal(err) is acceptable in main().