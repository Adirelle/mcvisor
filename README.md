# mcvisor

[![go 1.18](https://badgen.net/badge/go/1.18)](https://go.dev/)
[![main/lint](https://badgen.net/github/checks/Adirelle/mcvisor/main/lint)](https://github.com/Adirelle/mcvisor/actions/workflows/lint.yml)
[![main/test](https://badgen.net/github/checks/Adirelle/mcvisor/main/test)](https://github.com/Adirelle/mcvisor/actions/workflows/test.yml)
[![main/latest](https://badgen.net/github/checks/Adirelle/mcvisor/main/latest)](https://github.com/Adirelle/mcvisor/actions/workflows/latest.yml)
[![AGPL-3.0](https://badgen.net/github/license/Adirelle/mcvisor)](https://www.gnu.org/licenses/agpl-3.0.en.html)

mcvisor is a Minecraft server supervisor.
It launches and monitors a Minecraft server. It also starts a Discord bot
that can notify a channel of event and accepts commands.

# Planned features

(Non exhaustive list)

- General
  - [x] JSON configuration
  - [x] Resilient architecture based on [supervisor trees](http://www.jerf.org/iri/post/2930)
  - [x] Rotating file logging
  - [x] Console logging
  - [ ] CLI commands
  - [ ] Automatic configuration reload
  - [ ] Special signal handling (maybe, Linux only)
- Minecraft
  - [x] Server starting, stopping and restarting
  - [x] Automatic restarting
  - [x] Capture server logs
  - [x] Capture console output
  - [x] Monitor connectivity
  - [x] `!start`, `!stop`, `!restart` and `!shutdown` command to control the server
  - [x] `!online` command to list the players that are connected to the server
  - [x] `!status` command to show the server status
  - [ ] `!console` command to send commands to the server console
  - [ ] Scheduled restarts/scripts
  - [ ] Restart on unreachable status (maybe)
  - [ ] Disableable
- Discord Bot
  - [x] Automatic reconnection
  - [x] Accept commands
  - [x] User, channel and role permissions using Discord IDs
  - [x] Checks configuration on connection
  - [ ] Notifications in a given channel
- Web interface (maybe)
  - [ ] External authentification (which?)
  - [ ] Status display
  - [ ] Online player display
  - [ ] Server control
  - [ ] Partial log display
- Commands
  - [x] Extensible command system with permission checks
  - [x] `!help` command to list allowed commands
  - [x] `!perms` command to show command permissions

# License

Unless explicitly stated otherwise all files in this repository are licensed
under the [GNU Affero General Public License version 3 (AGPL-3)](https://www.gnu.org/licenses/agpl-3.0.en.html).
