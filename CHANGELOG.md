# Changelog

## 0.1.0 (2026-10-28)


### Features

* **frontend:** add CodeMirror 6 editor dependencies ([ac6e953](https://github.com/OLDAPS/open-ldap-studio/commit/ac6e9533f33e24b34c6c4b3d2a517a6d66619ae8))
* **frontend:** add CodeMirror 6 editor dependencies ([35d9656](https://github.com/OLDAPS/open-ldap-studio/commit/35d96564aaf9addb0bde7bd0549b80401a35a7d4))
* **frontend:** wire generated Go bindings and Vite hot reload ([a353d38](https://github.com/OLDAPS/open-ldap-studio/commit/a353d385c9977a3cee46b26e4d48f4973ba20702))
* **shell:** scaffold native Wails window and binding generation ([3d39b36](https://github.com/OLDAPS/open-ldap-studio/commit/3d39b36dd9c5137ed7592fa994096e966332c83d))


### Bug Fixes

* **ci:** isolate the pinned Wails binding generator ([e90ac1a](https://github.com/OLDAPS/open-ldap-studio/commit/e90ac1a03681cf4b2e688c3cf21f6cd5029727bf))
* **ci:** patch Go vulnerabilities and stabilize pruning tests ([f7a8c81](https://github.com/OLDAPS/open-ldap-studio/commit/f7a8c81ff167922b984f321ce8b4f7c09138e44f))
* **security:** update Go and Vite to patched releases ([efa88ae](https://github.com/OLDAPS/open-ldap-studio/commit/efa88aeee91be7b328cc33646d138710f3e8714c))


### Documentation

* add a getting-started guide for installing the application ([#483](https://github.com/OLDAPS/open-ldap-studio/issues/483)) ([2ede8c3](https://github.com/OLDAPS/open-ldap-studio/commit/2ede8c39f0fd781108905aa022d6a6330429c190))
* commit the spec-kit workspace and document spec-driven development ([#479](https://github.com/OLDAPS/open-ldap-studio/issues/479)) ([2548af7](https://github.com/OLDAPS/open-ldap-studio/commit/2548af76293c8346d7c44a9cb3d0ca786988d4ec))
* mark internal package skeleton complete ([62ffada](https://github.com/OLDAPS/open-ldap-studio/commit/62ffada58ae007800d2d31f0a39c4e6c6309a312))
* mark internal package skeleton complete ([7725e84](https://github.com/OLDAPS/open-ldap-studio/commit/7725e84be721e9f1573e1ff46cfca3a2b3be8d1f))


### Continuous Integration

* add pull request and Release Please release pipelines ([#468](https://github.com/OLDAPS/open-ldap-studio/issues/468)) ([0209bfb](https://github.com/OLDAPS/open-ldap-studio/commit/0209bfbe7d1bd908116618cb27ab71089190a3b2))
* build the .deb with dpkg-deb instead of nfpm ([#480](https://github.com/OLDAPS/open-ldap-studio/issues/480)) ([e4aa0af](https://github.com/OLDAPS/open-ldap-studio/commit/e4aa0af417f6c8b65c1dc1861c4c253d4bc2cef0))
* cache the Wails CLI instead of compiling it on every run ([#482](https://github.com/OLDAPS/open-ldap-studio/issues/482)) ([cc6a462](https://github.com/OLDAPS/open-ldap-studio/commit/cc6a46274a27430d67e73aba8b18a66aaf68263a))
* fold the quality gates into the CI workflow ([#481](https://github.com/OLDAPS/open-ldap-studio/issues/481)) ([3aa7cfe](https://github.com/OLDAPS/open-ldap-studio/commit/3aa7cfef1441854387492c1fcd32c222ded067d1))
* gate the whole tree on lint and add frontend coverage ([#472](https://github.com/OLDAPS/open-ldap-studio/issues/472)) ([3265aa4](https://github.com/OLDAPS/open-ldap-studio/commit/3265aa4d64ca3f81b426ea6c97235664b47e8801))
* package native installers and install-test them on every pull request ([#478](https://github.com/OLDAPS/open-ldap-studio/issues/478)) ([72edc64](https://github.com/OLDAPS/open-ldap-studio/commit/72edc6443919d44366507f15c6f0e3351871de1a))
* retry the NSIS install and fail where it fails ([#484](https://github.com/OLDAPS/open-ldap-studio/issues/484)) ([4b31e1f](https://github.com/OLDAPS/open-ldap-studio/commit/4b31e1fd729ba660a8cd230ced7c02c0d1966912))
