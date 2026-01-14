# Game Loop

### Server Browser

- Player launches the game and enters the server browser interface.
- **Dedicated Servers**: Players see a list of dedicated servers (similar to CS 1.6) displaying:
  - Map name
  - Current player count / max (up to 32 players per server)
  - Ping/latency
  - Game mode (wave defense)
  - Map rotation cycle
- Players can filter, sort, and refresh the server list.
- **Connect**: Player selects a server and joins it, entering the current map.
- **Other Features** (accessible from a separate main menu):
  - Character customization (skins, loadouts)
  - Store for purchasing permanent upgrades and cosmetics using Data currency
  - Shooting range to test weapons
  - Leaderboards and mission tracking

### Map

Each map contains **3 Points of Interest (POIs)** that must be defended sequentially. POIs are objectives such as generators, data terminals, or barricades.

#### Preparation Phase (60 seconds before each wave)

- Players start with a default amount of **Scrap** currency.
- **Building**: Place defensive structures (walls, turrets, traps) within build zones.
- **Shopping**: Buy weapons, ammo, and healing items from a temporary shop.
- **Skill Upgrades**: Purchase non‑persistent in‑game skill upgrades using skill points that last only for the current map.
- **Repair**: Repair damaged structures from previous waves (if any).
- **Strategy**: Team can allocate resources and assign roles.

#### Wave Phase

- Zombies, monsters, and vampires spawn from designated spawn points and pathfind toward the **active POI**.
- **POI Health**: Each POI has a health bar. If it reaches zero, the map fails.
- **Wave Composition**: Each wave increases in difficulty and introduces new enemy types.
- **POI Completion**: A POI is considered “done its job” after surviving a set number of waves (e.g., 3 waves). Then the next POI becomes active.
- **Map Completion**: After all 3 POIs have survived their waves, the map is completed.
- **Failure**: If any POI is destroyed, the map fails immediately.

#### Map Completed

- “Congratulations” screen shown with performance summary.
- **Rewards** distributed based on waves survived, kills, and team contribution.
- 15‑second timeout before the server automatically changes to the next map in rotation. Players may disconnect from the server during this timeout to return to the server browser.

#### Map Failed

- “Map Failed” screen with option to retry with the same team (if still on the same server).
- 15‑second timeout before the server automatically changes to the next map in rotation. Players may disconnect from the server during this timeout to return to the server browser.
- Players receive a reduced consolation reward (experience only).

### Flow Summary

1. **Server Browser** → Player selects a dedicated server
2. **Connect to Server** → Player joins the server and enters the current map
3. **Map** → For each wave: Preparation phase (60 seconds) → Wave phase (defend active POI) across 3 POIs × 3 waves total
4. **Map Transition** → After map completion or failure, a 15‑second timeout begins. During this timeout players may disconnect to return to the server browser. After the timeout, the server automatically changes to the next map in rotation. Players who remain on the server will continue to the next map.
