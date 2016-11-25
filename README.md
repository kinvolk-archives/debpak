# debpak - A helper to make a Flatpak from Debian packages

debpak generates a starting point for the `modules` section of a Flatpak json file.

It takes a path to the web page of a Debian package and gathers a list of dependencies.
One then needs to go through and remove any dependencies that are already found in the Sdk that is being used.

For example, to get the dependencies for vlc one would use the following:

`scraper https://packages.debian.org/jessie/web/vlc`

The output to stdout would like like such...

```
...
{
    "name": "fonts-freefont-ttf",
    "config-opts": "",
    "sources": [
      {
        "type": "archive",
        "url": "http://http.debian.net/debian/pool/main/v/vlc/vlc_2.2.4.orig.tar.xz",
        "sha256": "1632e91d2a0087e0ef4c3fb4c95c3c2890f7715a9d1d43ffd46329f428cf53be"
      }
    ]
  },
  {
    "name": "gcc-4.9-base",
    "config-opts": "",
    "sources": [
      {
        "type": "archive",
        "url": "http://http.debian.net/debian/pool/main/g/gcc-4.9/gcc-4.9_4.9.2.orig.tar.gz",
        "sha256": "861aa811d5f9e9ecf32d8195d2346fc434eba7e17330878ed3d876c49a32ec4e"
      }
    ]
  },
  {
    "name": "libgcc1",
    "config-opts": "",
    "sources": [
      {
        "type": "archive",
        "url": "http://http.debian.net/debian/pool/main/g/glibc/glibc_2.19.orig.tar.xz",
        "sha256": "746e52bb4fc9b2f30bcd33d415172a40ab56c5fff6c494052d31b0795593cc60"
      }
    ]
  },
...
```

Duplicates are filtered out. Ofter times, different Debian packages are derived
from the same "original" tarball.

If a package does not have an "original" tarball then the fields are left empty.
