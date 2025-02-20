<!-- markdownlint-disable -->
# Changelog

## [0.12.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.11.3...v0.12.0) (2025-02-20)


### Features

* spec updates handled and reflected by boost manager ([#89](https://github.com/google/kube-startup-cpu-boost/issues/89)) ([0da9e7e](https://github.com/google/kube-startup-cpu-boost/commit/0da9e7e96bf95ba42ff8ccff3d893762f23a4dbb))

## [0.11.3](https://github.com/google/kube-startup-cpu-boost/compare/v0.11.2...v0.11.3) (2024-12-20)


### Bug Fixes

* deps upgrade for vulnerability in golang.org/x/net/html ([#84](https://github.com/google/kube-startup-cpu-boost/issues/84)) ([3f92d37](https://github.com/google/kube-startup-cpu-boost/commit/3f92d371dda79459a0827e29796bfa3c72aa0c18))

## [0.11.2](https://github.com/google/kube-startup-cpu-boost/compare/v0.11.1...v0.11.2) (2024-12-13)


### Bug Fixes

* moved goreleaser manifest gen to avoid race cond ([#82](https://github.com/google/kube-startup-cpu-boost/issues/82)) ([a3a5248](https://github.com/google/kube-startup-cpu-boost/commit/a3a5248a5c6579a4166fad3014277f12ccb02c53))

## [0.11.1](https://github.com/google/kube-startup-cpu-boost/compare/v0.11.0...v0.11.1) (2024-12-13)


### Bug Fixes

* upgraded direct and transitive dependencies and go ([#80](https://github.com/google/kube-startup-cpu-boost/issues/80)) ([08fb82f](https://github.com/google/kube-startup-cpu-boost/commit/08fb82f3bb03e9db4ff65438f5610a3dc8ab5969))

## [0.11.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.10.1...v0.11.0) (2024-10-15)


### Features

* validation of required feature gate at start ([#72](https://github.com/google/kube-startup-cpu-boost/issues/72)) ([4b53f87](https://github.com/google/kube-startup-cpu-boost/commit/4b53f878b73d308ef2e39dce39f76e96d67b23a3))

## [0.10.1](https://github.com/google/kube-startup-cpu-boost/compare/v0.10.0...v0.10.1) (2024-10-10)


### Bug Fixes

* envtest version with available kubebuilder tools ([d3819e0](https://github.com/google/kube-startup-cpu-boost/commit/d3819e0440f7768b720ec274ff4c654ed6fa4083))
* improved logging when no resources are boosted ([bee8920](https://github.com/google/kube-startup-cpu-boost/commit/bee89206fa95247ef937a1d56019e66d4316db21))
* resource revert paniced when no limits were defined at all ([34cd968](https://github.com/google/kube-startup-cpu-boost/commit/34cd9686658f42978d6a9a46b6720c6ef548a15c))

## [0.10.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.9.0...v0.10.0) (2024-08-12)


### Features

* cpu resource limits removed during boost ([#59](https://github.com/google/kube-startup-cpu-boost/issues/59)) ([d4ffdad](https://github.com/google/kube-startup-cpu-boost/commit/d4ffdad779a83af0b7f5fac3c495fa6e6116f606))

## [0.9.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.8.1...v0.9.0) (2024-06-12)


### Features

* option to disable (def) HTTP2 for servers ([#53](https://github.com/google/kube-startup-cpu-boost/issues/53)) ([ad27fa9](https://github.com/google/kube-startup-cpu-boost/commit/ad27fa9ea855c9b17657bf4ab373285337995430))


### Bug Fixes

* adjusted log levels, logger names ans msgs ([#51](https://github.com/google/kube-startup-cpu-boost/issues/51)) ([f0ed258](https://github.com/google/kube-startup-cpu-boost/commit/f0ed258c06e22a1a5ece1a123b3afb117424e936))

## [0.8.1](https://github.com/google/kube-startup-cpu-boost/compare/v0.8.0...v0.8.1) (2024-06-11)


### Bug Fixes

* metrics endpoint not accesible from remote hosts ([#49](https://github.com/google/kube-startup-cpu-boost/issues/49)) ([f36dcea](https://github.com/google/kube-startup-cpu-boost/commit/f36dcea22111ec1b0e821741d4ed087468587a8d))

## [0.8.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.7.1...v0.8.0) (2024-06-07)


### Features

* metrics ([#47](https://github.com/google/kube-startup-cpu-boost/issues/47)) ([a051f05](https://github.com/google/kube-startup-cpu-boost/commit/a051f05ffb2e81d8dd57e57e73773321f101a0a5))

## [0.7.1](https://github.com/google/kube-startup-cpu-boost/compare/v0.7.0...v0.7.1) (2024-05-31)


### Bug Fixes

* kustomize tag typo ([#45](https://github.com/google/kube-startup-cpu-boost/issues/45)) ([992f00d](https://github.com/google/kube-startup-cpu-boost/commit/992f00d594305781e846e934817ba09036f6919f))

## [0.7.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.6.0...v0.7.0) (2024-05-24)


### Features

* structured logging ([#43](https://github.com/google/kube-startup-cpu-boost/issues/43)) ([f3d08c9](https://github.com/google/kube-startup-cpu-boost/commit/f3d08c90c74106c0d3ac5cd6a8e7e8fcff6516d1))

## [0.6.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.5.0...v0.6.0) (2024-05-21)


### Features

* status and conditions for StartupCPUBoost ([#39](https://github.com/google/kube-startup-cpu-boost/issues/39)) ([8678a00](https://github.com/google/kube-startup-cpu-boost/commit/8678a00d3e8e2fbb3362c6e35be1b419cd0e437d))
* time policy based pod revert done in parallel ([#41](https://github.com/google/kube-startup-cpu-boost/issues/41)) ([e04806a](https://github.com/google/kube-startup-cpu-boost/commit/e04806ae357001a5978c6aad695597c53d0cc0ef))

## [0.5.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.4.1...v0.5.0) (2024-03-15)


### Features

* adding config abstraction ([#31](https://github.com/google/kube-startup-cpu-boost/issues/31)) ([ac47461](https://github.com/google/kube-startup-cpu-boost/commit/ac47461f23d3d59cc93bed4b0ef3a1ee59fe3af6))
* zap log level as environment variable ([#35](https://github.com/google/kube-startup-cpu-boost/issues/35)) ([d019b7a](https://github.com/google/kube-startup-cpu-boost/commit/d019b7ae5bfbee017a4a155a42fb28a4fccb33a8))

## [0.4.1](https://github.com/google/kube-startup-cpu-boost/compare/v0.4.0...v0.4.1) (2024-03-06)


### Bug Fixes

* pod webhook cfg blocks scheduling on failure ([#28](https://github.com/google/kube-startup-cpu-boost/issues/28)) ([1f48f53](https://github.com/google/kube-startup-cpu-boost/commit/1f48f5337ab23af6b7421df95f2ebc99111c1b17))

## [0.4.0](https://github.com/google/kube-startup-cpu-boost/compare/v0.3.0...v0.4.0) (2024-02-09)


### Features

* support running in non-default namespace ([#20](https://github.com/google/kube-startup-cpu-boost/issues/20)) ([f3cdc46](https://github.com/google/kube-startup-cpu-boost/commit/f3cdc46d262c18d591dd7d565655060d0d10ee89))

## 0.3.0 (Dec 29, 2023)

FEATURES:

* Possibility to set fixed value in a CPU boost target [#18](https://github.com/google/kube-startup-cpu-boost/pull/18)

## 0.2.0 (Dec 8, 2023)

FEATURES:

* POD mutating Webhook reflects container resize policy [#14](https://github.com/google/kube-startup-cpu-boost/pull/14)
* Introduced StartupCPUBoost validating webhook [#16](https://github.com/google/kube-startup-cpu-boost/pull/16)

## 0.1.0 (Dec 4, 2023)

FEATURES:

* Introduced container resource policies [#13](https://github.com/google/kube-startup-cpu-boost/pull/13)

## 0.0.2 (Nov 15, 2023)

FEATURES:

* Major refactor + introduction of duration policies [#12](https://github.com/google/kube-startup-cpu-boost/pull/12)

## 0.0.1 (Oct 24, 2023)

PoC version of the solution
