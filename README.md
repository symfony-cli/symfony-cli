<p align="center"><a href="https://symfony.com" target="_blank">
    <img src="https://symfony.com/logos/symfony_black_02.svg">
</a></p>

The [Symfony binary][1] is a must-have tool when developing Symfony applications
on your local machine. It provides:

* The best way to [create new Symfony applications][2];
* A powerful [local web server][3] to develop your projects with support for [TLS certificates][4];
* A tool to [check for security vulnerabilities][5];
* Managed [Symfony-specific diagnostics][12] for local development and CI;
* Seamless integration with [Upsun (formerly Platform.sh)][6].

Installation
------------

Read the installation instructions on [symfony.com][7].

Signature Verification
----------------------

Symfony release artifacts are signed using [cosign][8], which is part of
[sigstore][9]. Download an artifact and its matching Sigstore bundle, then
verify it as follows:

```console
$ cosign verify-blob \
    --bundle symfony-cli_linux_amd64.tar.gz.sigstore.json \
    --certificate-identity-regexp='^https://github\.com/symfony-cli/symfony-cli/\.github/workflows/releaser\.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$' \
    --certificate-oidc-issuer='https://token.actions.githubusercontent.com' \
    symfony-cli_linux_amd64.tar.gz
Verified OK
```

The signatures use Sigstore's [keyless signing][10] method.

Security Issues
---------------

If you discover a security vulnerability, please follow our [disclosure procedure][11].

Sponsorship [<img src="https://assets.cloudsmith.media/images/cloudsmith-logo-light.svg" width="250" align="right" />](https://cloudsmith.io/)
-----------

Package repository hosting is graciously provided by
[cloudsmith](https://cloudsmith.io/). Cloudsmith is the only fully hosted,
cloud-native, universal package management solution, that enables your
organization to create, store and share packages in any format, to any place,
with total confidence. We believe there’s a better way to manage software
assets and packages, and they're making it happen!

[1]: https://symfony.com/download
[2]: https://symfony.com/doc/current/setup.html#creating-symfony-applications
[3]: https://symfony.com/doc/current/setup/symfony_server.html
[4]: https://symfony.com/doc/current/setup/symfony_server.html#enabling-tls
[5]: https://symfony.com/doc/current/setup.html#security-checker
[6]: https://symfony.com/cloud
[7]: https://symfony.com/download
[8]: https://github.com/SigStore/cosign
[9]: https://www.sigstore.dev/
[10]: https://docs.sigstore.dev/cosign/signing/signing_with_blobs/
[11]: https://symfony.com/security
[12]: https://github.com/symfony/language-tools/blob/main/docs/features/headless-diagnostics.rst
