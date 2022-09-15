## Unreleased

## 0.7.0 (September 15, 2022)

Improvements:
* Updated golang dependencies [GH-85](https://github.com/hashicorp/vault-plugin-database-oracle/pull/85)
  * golang.org/x/crypto@v0.0.0-20220314234659-1baeb1ce4c0b
  * golang.org/x/sys@v0.0.0-20220412211240-33da011f77ad
  * golang.org/x/net@v0.0.0-20211209124913-491a49abca63

## 0.6.1 (March 23, 2022)

* Re-release of 0.6.0

## 0.6.0 (March 23, 2022)

Features:
* Add support for plugin multiplexing [[GH-74](https://github.com/hashicorp/vault-plugin-database-oracle/pull/74)]

## 0.5.1 (December 16, 2021)

* Remove vendored dependencies in the repository. This change should be transparent for plugin uses. [[GH-69](https://github.com/hashicorp/vault-plugin-database-oracle/pull/69)]

## 0.5.0 (December 16, 2021)

Features:
* Add ability to fully customize revocation statements [[GH-62](https://github.com/hashicorp/vault-plugin-database-oracle/pull/62)]

## 0.4.2 (August 23, 2021)

Improvements:
* Improved session killing logic for RAC clusters & local databases [[GH-60](https://github.com/hashicorp/vault-plugin-database-oracle/pull/60)]

## 0.4.1 (May 18, 2021)

Bug Fixes:
* Updates dependent library which removed a number of memory leaks [[GH-53](https://github.com/hashicorp/vault-plugin-database-oracle/pull/53)]

## 0.4.0 (March 22, 2021)

Features:
* Adds the ability to customize how usernames are generated via username templates [[GH-47](https://github.com/hashicorp/vault-plugin-database-oracle/pull/47)]

## 0.3.1 (May 18, 2021)

Bug Fixes:
* Updates dependent library which removed a number of memory leaks [[GH-56](https://github.com/hashicorp/vault-plugin-database-oracle/pull/56)]

## 0.2.2 (May 18, 2021)

Bug Fixes:
* Updates dependent library which removed a number of memory leaks [[GH-55](https://github.com/hashicorp/vault-plugin-database-oracle/pull/55)]
