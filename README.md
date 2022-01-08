<p align="center"><a href="https://symfony.com" target="_blank">
    <img src="https://symfony.com/logos/symfony_black_02.svg">
</a></p>

The [Symfony binary][1] is a must-have tool when developing Symfony applications
on your local machine. It provides:

* The best way to [create new Symfony applications][2];
* A powerful [local web server][3] to develop your projects with support for [TLS certificates][4];
* A tool to [check for security vulnerabilities][5];
* Seamless integration with [Platform.sh][6].

Installation
------------

### Binaries

To install Symfony CLI, please download the [appropriate version](https://github.com/symfony-cli/symfony-cli/releases),
unarchive the files, and move the binary somewhere under your path.

Downloading a binary is quick and simple, but upgrading is manual: download the latest version
and replace the binary by the new one.

To automatically get updates when available, see below.

### Linux

You can download the `.deb`, `.rpm`, and `.apk` packages from the
[release page](https://github.com/symfony-cli/symfony-cli/releases).

You can also use GoFish (see below).

### macOS

Use homebrew to install and get automatic updates:

    brew install symfony-cli/tap/symfony-cli

### Windows

Use Scoop:

    scoop bucket add symfony-cli https://github.com/symfony-cli/scoop-bucket.git
    scoop install symfony-cli

You can also use GoFish (see below).

### GoFish

On Linux and Windows, you ocan use GoFish:

    gofish rig add https://github.com/symfony-cli/fish-food
    gofish install github.com/symfony-cli/fish-food/symfony-cli

Security Issues
---------------

If you discover a security vulnerability, please follow our [disclosure procedure][7].

[1]: https://symfony.com/download
[2]: https://symfony.com/doc/current/setup.html#creating-symfony-applications
[3]: https://symfony.com/doc/current/setup/symfony_server.html
[4]: https://symfony.com/doc/current/setup/symfony_server.html#enabling-tls
[5]: https://symfony.com/doc/current/setup.html#security-checker
[6]: https://symfony.com/cloud
[7]: https://symfony.com/security
