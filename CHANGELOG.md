<!-- markdownlint-disable -->
# Changelog

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
