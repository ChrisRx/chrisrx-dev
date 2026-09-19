---
title: Neovim Nix module
date: 2026-09-17T09:30:00Z
---

![My Neovim configuration](/assets/images/neovim.png)

I love Neovim as a programming IDE, but configuration can be a bit laborious. This is especially true as I need it on more and more computers (work lappy, personal lappy, home server, living room computer, et al) where improvements and other drift become difficult to keep in sync. I've been using NixOS for a couple years now and the solution for management was [Nixvim](https://nix-community.github.io/nixvim/), but keeping things updated across multiple systems was still a chore. My work laptop came with [Pop!_OS](https://system76.com/pop) and has home-manager installed and my other computers are are all running NixOS so the problem was almost completely solved. Now I'm exporting my neovim configuration as a NixOS and Home Manager module, so it can be imported where ever I need to use it. I can even run it ad hoc on any system with the Nix package manager installed:

```shell
nix run github:ChrisRx/nixos-config#neovim
```

Check out my multi-host [NixOS config](https://github.com/ChrisRx/nixos-config).
