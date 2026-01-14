# Ai zombie defence

## Overview

AI Zombie Defence is an online multiplayer co-op game via dedicated servers where players defend points of interest against waves of zombies, monsters, and vampires. The gameplay combines wave‑based survival (similar to Killing Floor and Call of Duty Zombies) with tower‑defense mechanics inspired by Yet Another Zombie Defense. The core loop consists of:

1. **Main Menu**: Customize character (skins, loadouts), view cosmetic progression, and browse servers.
2. **Server Browser**: Browse dedicated servers running different maps in rotation; players can connect to any server.
3. **Map**: Defend points of interest through preparation and wave phases.
4. **Rewards**: Earn currency, loot, and experience based on performance.
5. **Progression**: Level up, unlock cosmetics, and prestige.

[Game loop](./game-loop.md)

## Progression System

- **Cosmetic Progression**: Earn experience (XP) from killing enemies and completing maps. Each level unlocks new cosmetic items (skins, emotes, etc.).
- **In-Game Upgrades**: During each map, players earn skill points that can be spent on temporary upgrades (e.g., +10% health, faster reload, building durability). These upgrades reset at the end of the map.
- **Weapon Access**: All weapons are available in the map's shop; players purchase them using scrap earned during the match.
- **Building Access**: All defensive structures are available in the map's shop; players purchase them using scrap earned during the match.
- **Prestige**: After reaching max level, players can prestige for exclusive cosmetics.

## Rewards Economy

- **Scrap**: Common currency earned from killing enemies. Used for in‑map purchases (weapons, ammo, buildings).
- **Data**: Rare currency earned from completing maps. Used for cosmetics.
- **Loot Drops**: Special enemies have a chance to drop random weapon skins or cosmetic items.
- **Shop**: In‑map shop sells consumables and temporary gear. Cosmetics are purchased from a main menu store using Data currency.

## Player Interactions

- **Server Browser**: Browse dedicated servers running different maps in rotation; players can connect to any server.
- **Co‑op**: Up to 32 players per map via dedicated servers. Shared objectives, revive mechanics (downed players can be revived by teammates).
- **Social Features**: Text and voice chat, ping system for non‑verbal communication, friend lists.

## Combat Mechanics

**Inspirations**: Combat takes inspiration from Killing Floor and Call of Duty Zombies wave‑based survival, while tower‑defense mechanics are inspired by Yet Another Zombie Defense.

- **Weapon Types**:
  - Ranged: Assault rifles, shotguns, sniper rifles, pistols.
  - Melee: Bat, axe, knife (fast, low damage).
  - Throwables: Grenades, molotovs, flashbangs.
- **Enemy Types**:
  - Walker: Slow, high health.
  - Runner: Fast, low health.
  - Tank: Very high health, deals heavy damage.
  - Spitter: Ranged acid attack.
  - Vampire: Drains player health, can teleport short distances.
- **AI Behavior**: Enemies pathfind to the active POI, attacking players who block their path. Special enemies may prioritize players.
- **Building Mechanics**:
  - Walls: Block enemy movement, can be repaired.
  - Turrets: Auto‑target enemies in range.
  - Traps: Slow or damage enemies that pass over them.
- **Health System**: Players have health and optional armor. Healing items (medkits) restore health over time.

## Non‑Functional Requirements

### Performance

- **Server Performance**: Support up to 32 concurrent players per map with stable 30 Hz server tick rate
- **Network Latency**: Game remains playable with up to 150 ms ping; client‑side prediction and server reconciliation for smooth gameplay
- **Client Performance**: Target 60 FPS on mid‑range hardware (e.g., GTX 1060, 8 GB RAM)
- **Loading Times**: Maps load within 30 seconds on average broadband connections

### Scalability

- **Horizontal Scaling**: Dedicated server architecture allows spinning up new servers as player count grows
- **Server Browser**: Capable of listing hundreds of active servers with real‑time player counts and ping data
- **Database**: Support thousands of player accounts and cosmetic item unlocks with sub‑second read/write latency

### Reliability & Availability

- **Uptime**: Dedicated servers target 99 % uptime; automated health checks and restart mechanisms
- **Reconnect**: Players can rejoin the same match if disconnected (graceful degradation)
- **Data Persistence**: Player progression (cosmetics, currency) saved reliably with regular backups

### Security

- **Anti‑Cheat**: Server‑side validation of player actions; detect and prevent common exploits
- **Authentication**: Optional account system with secure credential storage
- **DDoS Mitigation**: Basic protection against volumetric attacks on server infrastructure
- **Input Validation**: All client inputs validated server‑side before affecting game state

### Usability

- **Learnability**: New players understand core loop (prepare → defend → progress) within first session
- **Customization**: Configurable key bindings and sensitivity settings
- **Accessibility**: Color‑blind modes, subtitle support for audio cues, adjustable UI scale

### Compatibility

- **Cross‑Platform**: Godot cross‑platform compilation for Windows, Linux, macOS clients; no platform‑specific development overhead
- **Servers**: Linux‑based dedicated servers
- **Hardware**: Minimum specifications documented for predictable performance
- **Input Devices**: Support for keyboard/mouse and common game controllers

### Maintainability

- **Modular Architecture**: Code organized for independent updates to game systems (combat, economy, UI)
- **Monitoring**: Server logging and metrics for operational visibility
- **Deployment**: Streamlined process for deploying new maps, balance changes, and cosmetic content

### Technical Architecture Constraints

- **Backend Services**: Golang with sqlc and SQLite for account, progression, and match‑history services
- **Game Client/Server**: Godot (GDScript) for game logic; C# only for performance‑critical subsystems that require database access from the server
- **Code Language Strategy**: Prefer GDScript for gameplay code; C# for server‑side database‑heavy operations; maintain cross‑platform compatibility through Godot’s toolchain

### Legal & Compliance

- **Privacy**: Compliance with GDPR/COPPA where applicable; clear privacy policy
- **Terms of Service**: Explicit rules against cheating, harassment, and inappropriate content

## Notes & Open Questions

- How many waves per POI? (3 waves per POI, 9 waves total per map)
- Should players keep their purchased weapons between maps? (No – weapons are lost at the end of each map)
- What is the maximum players per server? (32 players)
