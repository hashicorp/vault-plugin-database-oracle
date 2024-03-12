## Unreleased
IMPROVEMENTS:
* Update dependencies [GH-147](https://github.com/hashicorp/vault-plugin-database-oracle/pull/147)
  * `github.com/hashicorp/vault/api` v1.11.0 -> v1.12.0
  * `github.com/hashicorp/vault/sdk` v0.10.2 -> v0.11.0
* Bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.3 [GH-148](https://github.com/hashicorp/vault-plugin-database-oracle/pull/148)
* Bump github.com/jackc/pgx/v4 from 4.18.1 to 4.18.2 [GH-149](https://github.com/hashicorp/vault-plugin-database-oracle/pull/149)

## 0.9.0 (February 2, 2024)
IMPROVEMENTS:
* Update dependencies [GH-144](https://github.com/hashicorp/vault-plugin-database-oracle/pull/144)
  * github.com/hashicorp/vault/api v1.9.2 -> v1.11.0
  * github.com/hashicorp/vault/sdk v0.9.2 -> v0.10.2
* Bump github.com/go-jose/go-jose/v3 from 3.0.0 to 3.0.1 [GH-138](https://github.com/hashicorp/vault-plugin-database-oracle/pull/138)
* Bump golang.org/x/crypto from 0.12.0 to 0.17.0 [GH-140](https://github.com/hashicorp/vault-plugin-database-oracle/pull/140)
* Bump github.com/docker/docker from 23.0.4+incompatible to 23.0.8+incompatible [GH-142](https://github.com/hashicorp/vault-plugin-database-oracle/pull/142)
* Bump github.com/opencontainers/runc from 1.1.6 to 1.1.12 [GH-143](https://github.com/hashicorp/vault-plugin-database-oracle/pull/143)
* Update dependencies [GH-145](https://github.com/hashicorp/vault-plugin-database-oracle/pull/145)
  * Update Go from 1.20.7 to 1.21.6
  * Update Oracle Instant Client from 19.20 to 19.22

## 0.8.3 (August 31, 2023)
IMPROVEMENTS:
* Update dependencies [GH-129](https://github.com/hashicorp/vault-plugin-database-oracle/pull/129)
  * `github.com/hashicorp/vault/api` v1.9.1 -> v1.9.2
  * `github.com/hashicorp/vault/sdk` v0.9.0 -> v0.9.2
* Update dependencies [GH-130](https://github.com/hashicorp/vault-plugin-database-oracle/pull/130)
  * Update Go to 1.20.7
  * Update Oracle Instant Client to 19.20
* Update dependencies [GH-131](https://github.com/hashicorp/vault-plugin-database-oracle/pull/131)
  * golang.org/x


## 0.8.2 (May 25, 2023)

Improvements:

* Update dependencies [GH-123](https://github.com/hashicorp/vault-plugin-database-oracle/pull/123)
  * github.com/hashicorp/vault/api v1.9.1
  * github.com/hashicorp/vault/sdk v0.9.0
  * github.com/ory/dockertest/v3 v3.10.0

## 0.8.1 (March 24, 2023)

Improvements:

* Update dependencies [GH-115](https://github.com/hashicorp/vault-plugin-database-oracle/pull/115)
  * github.com/opencontainers/runc v1.1.4
* Update dependencies
  * Update Go to 1.20.2 [GH-114](https://github.com/hashicorp/vault-plugin-database-oracle/pull/114)
* Update dependencies [GH-109](https://github.com/hashicorp/vault-plugin-database-oracle/pull/109)
  * github.com/hashicorp/vault/api v1.9.0
  * golang.org/x/crypto v0.5.0
* Update dependencies [GH-108](https://github.com/hashicorp/vault-plugin-database-oracle/pull/108)
  * github.com/hashicorp/vault/sdk v0.8.1
  * github.com/hashicorp/go-kms-wrapping v2.0.7
* Update dependencies [GH-107](https://github.com/hashicorp/vault-plugin-database-oracle/pull/107)
  * golang.org/x/net v0.7.0
  * golang.org/x/sys v0.5.0
  * golang.org/x/text v0.7.0

## 0.8.0 (February 13, 2023)

Improvements:
* Update dependencies [GH-105](https://github.com/hashicorp/vault-plugin-database-oracle/pull/105)
  * Update Oracle Instant Client to 19.18
  * Update Go to 1.19.5
* Update dependencies [GH-93](https://github.com/hashicorp/vault-plugin-database-oracle/pull/93)
  * github.com/hashicorp/vault/api v1.8.3
  * github.com/hashicorp/vault/sdk v0.7.0
  * github.com/ory/dockertest/v3 v3.9.1

## 0.7.0 (September 16, 2022)

Improvements:
* Updated golang dependencies [GH-85](https://github.com/hashicorp/vault-plugin-database-oracle/pull/85)
  * golang.org/x/crypto@v0.0.0-20220314234659-1baeb1ce4c0b
  * golang.org/x/sys@v0.0.0-20220412211240-33da011f77ad
  * golang.org/x/net@v0.0.0-20220909164309-bea034e7d591 [GH-89](https://github.com/hashicorp/vault-plugin-database-oracle/pull/89)
* Update Go to 1.19.1 [GH-89](https://github.com/hashicorp/vault-plugin-database-oracle/pull/89)
* Update Vault dependencies [GH-89](https://github.com/hashicorp/vault-plugin-database-oracle/pull/89)
  * `github.com/hashicorp/vault/api v1.7.2`
  * `github.com/hashicorp/vault/sdk v0.5.3`

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
