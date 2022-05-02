# mcvisor

[![go 1.18](https://badgen.net/badge/go/1.18)](https://go.dev/)
[![Lint](https://github.com/Adirelle/mcvisor/actions/workflows/lint.yml/badge.svg)](https://github.com/Adirelle/mcvisor/actions/workflows/lint.yml)
[![Test](https://github.com/Adirelle/mcvisor/actions/workflows/test.yml/badge.svg)](https://github.com/Adirelle/mcvisor/actions/workflows/test.yml)
[![Latest binaries](https://github.com/Adirelle/mcvisor/actions/workflows/latest.yml/badge.svg)](https://github.com/Adirelle/mcvisor/actions/workflows/latest.yml)
[![AGPL-3.0](https://badgen.net/github/license/Adirelle/mcvisor)](https://www.gnu.org/licenses/agpl-3.0.en.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/Adirelle/mcvisor)](https://goreportcard.com/report/github.com/Adirelle/mcvisor)

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
- Minecraft
  - [x] Server starting, stopping and restarting
  - [x] Automatic restarting
  - [x] Capture server logs
  - [x] Capture console output
  - [x] Monitor connectivity
  - [x] `!start`, `!stop`, `!restart` and `!shutdown` command to control the server
  - [x] `!online` command to list the players that are connected to the server
  - [x] `!status` command to show the server status
  - [x] `!console` command to send commands to the server console
  - [ ] Preconfigured jobs
  - [ ] Scheduled restarts/scripts
  - [ ] Restart on unreachable status (maybe)
- Discord Bot
  - [x] Automatic reconnection
  - [x] Accept commands
  - [x] User, channel and role permissions using Discord IDs
  - [x] Checks configuration on connection
  - [x] Notifications in a given channel
- Commands
  - [x] Extensible command system with permission checks
  - [x] `!help` command to list allowed commands

# License

Unless explicitly stated otherwise all files in this repository are licensed
under the [GNU Affero General Public License version 3 (AGPL-3)](https://www.gnu.org/licenses/agpl-3.0.en.html).
