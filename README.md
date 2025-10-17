# Peridot

Peridot is a lightweigh, modular dotfiles manager written in Go. It helps you organize, deploy and check the status of your dotfiles using a simple module-based system.

---

## Features

- Initialize a dotfiles directory that stores both peridot's state/config and your dotfiles.
- Create and manage modules for different configurations (e.g. nvim, hyprland, git...).
- Deploy dotfiles as symlinks safely, with clear control over peridot's behavior on collision.
- Treat your module's dotfiles as templates without worrying about intermediate "dirty" files.
- Display the current state of your dotfiles directory with a clear tree view.
- Simulate the deployment of a module's dotfiles to freely test peridot's behavior before making any changes to the filesystem.

---

## Installation

Make sure you have **Go 1.24.6+** installed.

```bash
go install github.com/mermonia/peridot@v0.1.0
```

The binary will be installed in $GOPATH/bin (usually ~/go/bin).

---

## Usage

### Initialize your dotfiles directory

```bash
peridot init
```
- Initializes the current dir (or the one specified by the corresponding environment variable) as a dotfiles dir.
- Creates a .peridot dir inside it for peridot's internal files (temp files, state...).

### Add a new module

```bash
peridot add nvim
```
- Creates a new module named nvim.
- Adds it to peridot's list of managed modules.

### Configure the module (optional)

Creating a new module dir also creates a default configuration inside it. While peridot's default configuration will work for most users, there is a lot of room for customization:
- Root to which the module's dotfiles will be deployed.
- Dependencies to binaries/other modules to be checked before deployment.
- Hooks to be executed before/after deployment.
- Variables for the rendering of template files.
- Other OS conditions.

### Add your dotfiles

The contents of your module should mimic the structure of your own filesystem (similarly to other tools like GNU Stow). The default starting point for a module is "~", but you can change it to your liking in the module's configuration file or on deployment via the --root flag.

### Deploy your dotfiles

```bash
peridot deploy nvim
```
- Renders the template files to intermediate files in the .peridot dir
- Creates symlinks to the rendered files in the actual filesystem

Collisions with existing files in the filesystem can be managed via flags (--adopt, --overwrite).

You might want to simulate the deployment of a module before making any actual changes to the filesystem:

```bash
peridot deploy nvim --simulate
```
- Does not make any changes to either the filesystem or the dotfiles dir.
- Reports any changes it would make without the --simulate flag.

### Check the status of your dotfiles

```bash
peridot status
```
- Prints the current state of the dotfiles dir.

An example output:
```console
foo@bar:~/myDotfiles$ peridot status
.
├── ✓ nvim - deployed and up to date
│   └── .myconfig
│       └── nvim
│           ├── lua
│           │   └── mermonia
│           │       ├── ✓ remap.lua <- /home/umbraslay/.myconfig/nvim/lua/mermonia/remap.lua
│           │       ├── ✓ set.lua <- /home/umbraslay/.myconfig/nvim/lua/mermonia/set.lua
│           │       └── ✓ init.lua <- /home/umbraslay/.myconfig/nvim/lua/mermonia/init.lua
│           └── ✓ init.lua <- /home/umbraslay/.myconfig/nvim/init.lua
└── ○ hyprland - not deployed
```

## Configuration

Each module's behavior can be customized by editing the **module.toml** file inside each module directory. This file is automatically created when adding a module.

## License

[MIT](https://choosealicense.com/licenses/mit/)
