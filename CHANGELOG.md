<!-- markdownlint-disable -->
# Changelog

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
