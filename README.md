# What?

A Minecraft server supervisor that notifies Discord channels and accepts commands from them.

# Why not a Minecraft mod?

Because a mod is useless when the server is down. The supervisor is independant of the server and monitors it.
It is still there if the server goes down, can restart it, etc...

# Planned features

(Non exhaustive list)

- General
  - [x] JSON configuration
  - [x] Rotating file logging
  - [x] Console logging
- Minecraft
  - [x] Server starting, stopping and restarting
  - [x] Automatic restarting
  - [x] Capture server logs
  - [x] Capture console output
  - [x] Monitor connectivity
  - [x] `!start`, `!stop`, `!restart` and `!shutdown` command to control the server
  - [x] `!online` command to list the players that are connected to the server
  - [x] `!status` command to show the server status
  - [ ] `!console` command to send command to the serve console
  - [ ] Planned restarts/commands
  - [ ] Restart on unreachable status (?)
- Discord Bot
  - [x] Automatic reconnection
  - [x] Accept commands
  - [x] User, channel and role permissions using Discord IDs
  - [x] Checks server membership and channels on connection
  - [ ] Notifications in a given channel
- Commands
  - [x] Extendable command system with permission checks
  - [x] `!help` command to list allowed commands
  - [x] `!perms` command to show command permissions

# License

Unless explicitly stated otherwise all files in this repository are licensed under the GNU Affero General Public License version 3 (AGPL-3).
