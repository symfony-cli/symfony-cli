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

Read the installation instructions on [symfony.com][7].

Signature Verification
----------------------

Symfony binaries are signed using [cosign][8], which is part of [sigstore][9].
Signatures can be verified as follows (OS and architecture omitted for clarity):

```console
$ COSIGN_EXPERIMENTAL=1 cosign verify-blob --signature symfony-cli.sig symfony-cli
tlog entry verified with uuid: "2b7ca2bfb7ee09114a15d60761c2a0a8c97f07cc20c02e635a92ba137a08a6de" index: 1261963
Verified OK
```

The above uses the (currently experimental) [keyless signing][10] method.
Alternatively, one can verify the signature by also providing the certificate:

```console
$ cosign verify-blob --cert symfony-cli.pem --signature symfony-cli.sig symfony-cli
Verified OK
```

Security Issues
---------------

If you discover a security vulnerability, please follow our [disclosure procedure][11].

Sponsorship [<img src="https://cloudposse.com/wp-content/uploads/2020/10/cloudsmith.svg" width="250" align="right" />](https://cloudsmith.io/)
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
[10]: https://github.com/sigstore/cosign/blob/main/KEYLESS.md
[11]: https://symfony.com/security
