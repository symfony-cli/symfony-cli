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

### Linux

Depending on your Linux distribution, use the Symfony `apt` repository:

    echo 'deb [trusted=yes] https://repo.symfony.com/apt/ /' | sudo tee /etc/apt/sources.list.d/symfony-cli.list
    sudo apt update
    sudo apt install symfony-cli

Or the Symfony `yum` repository:

    echo '[symfony-cli]
    name=Symfony CLI
    baseurl=https://repo.symfony.com/yum/
    enabled=1
    gpgcheck=0' | sudo tee /etc/yum.repos.d/symfony-cli.repo
    sudo yum install symfony-cli

You can also download the `.deb`, `.rpm`, and `.apk` packages directly from the
[release page][7].

You can also use GoFish (see below).

### macOS

Use homebrew:

    brew install symfony-cli/tap/symfony-cli

### Windows

Use Scoop:

    scoop bucket add symfony-cli https://github.com/symfony-cli/scoop-bucket.git
    scoop install symfony-cli

You can also use GoFish (see below).

### GoFish

On Linux and Windows, you can use GoFish:

    gofish rig add https://github.com/symfony-cli/fish-food
    gofish install github.com/symfony-cli/fish-food/symfony-cli

### Binaries

You can also download Symfony CLI binaries from the [release page][7],

Unarchive the files, and move the binary somewhere under your path.

Downloading a binary is quick and simple, but upgrading is manual: download the latest version
and replace the binary by the new one.

Security Issues
---------------

If you discover a security vulnerability, please follow our [disclosure procedure][8].

[1]: https://symfony.com/download
[2]: https://symfony.com/doc/current/setup.html#creating-symfony-applications
[3]: https://symfony.com/doc/current/setup/symfony_server.html
[4]: https://symfony.com/doc/current/setup/symfony_server.html#enabling-tls
[5]: https://symfony.com/doc/current/setup.html#security-checker
[6]: https://symfony.com/cloud
[7]: https://github.com/symfony-cli/symfony-cli/releases/latest
[8]: https://symfony.com/security
