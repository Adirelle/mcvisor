# What?

A Minecraft server supervisor that notifies Discord channels and accepts commands from them.

# Why not a Minecraft mod?

Because a mod is useless when the server is down. The supervisor is independant of the server and monitors it.
It is still there if the server goes down, can restart it, etc...

# Features

Most configuration is done through a JSON file.

## Minecraft server supervision

* Start, stop and restart the server on demand.
* Monitor the server process.
* (maybe) Monitor the network status.
* Autostart the server if configured so.
* Restart the server when it goes down unexpectedly, eventually with a delay between each tries and a maximum number of tries.

## Discord integration

* Notify channels of various events (server status change, console messages, ...).
* Accept commands.
* Per-command access control, leveraging channels/roles/users.

### Commands

Grouped by permissions:

* `control`:
  * `!start`: start the server
  * `!stop [when]`: stop the server at a given time (possibly "now", "10:00", "5 minutes", ...)
  * `!restart [when]`:  restart the server at a given time (possibly "now")
* `query`:
  * `!status`: display server status
  * `!online`: list online players
* `console`:
  * `!console [text]`: send text to the console

## Other?

* A command to download logs?
* Scheduled restarts?
* CLI to send commands?
