# KubeFox CLI (Fox)

[![Go Report Card](https://goreportcard.com/badge/github.com/xigxog/kubefox-cli)](https://goreportcard.com/report/github.com/xigxog/kubefox-cli)

CLI for interacting with the KubeFox platform.

## Synopsis

Fox 🦊 is a CLI for interacting with the KubeFox platform. You can use it to
create, build, validate, and manage your KubeFox Components, Applications, and
Systems.

## Example Usage

> Note: this assumes an already configured KubeFox installation.

### Initialization

1. Create a new repository in your GitHub organization. This will be the repo
   for your new system.
2. run `git clone <your repo here>` to clone the repo to your machine.
3. run `cd <your repo name>` to change into your git repos directory.
4. run `fox init`

> At this point your system will be setup to work with KubeFox, added to your
> KubeFox Platform, and checked into git.

### Publish, Deploy, Release

1. Check in all your application code in the components directory, and necessary
   information in your application yaml file
2. Create configuration and environment files, and apply them using `fox apply
-f <path/to/file>`
3. Tag your config and environments using `fox create tag <env|config/name>
--tag <semver>--id <id of object>`
4. run `fox publish -t <semver>` to build your application images and tag your
   System. This will update your system in KubeFox with this System Ref. NOTE:
   This command requires access to a docker daemon.
5. run `fox create deployment --system <system name> -t <semver of system ref>`
   to deploy your specific System Reference.
6. run `fox create release --systemref <path/to/sysref> --environment
<path/to/env>` to release your applications to your users.

## NOTES

> See docs folder for full documentation on the CLI.
