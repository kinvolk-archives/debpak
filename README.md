# debpak - A helper to make a Flatpak from Debian packages

debpak generates a starting point for the `modules` section of a Flatpak json file.

It takes a path to the web page of a Debian package and gathers a list of dependencies.
One then needs to go through and remove any dependencies that are already found in the Sdk that is being used.

For example, to get the dependencies for vlc one would use the following:

`scraper -pkg vlc`

The output to stdout would like like such...

```
...
{
    "name": "libvcdinfo0",
    "config-opts": "",
    "sources": [
      {
        "type": "file",
        "url": "http://ftp.us.debian.org/debian/pool/main/v/vlc/vlc-nox_2.2.4-1~deb8u1_amd64.deb",
        "sha256": "56b0a9a2e3009515f1654a7fc960e528cbb78c054260c12addbd613fd13b9af9"
      }
    ]
  },
  {
    "name": "libvlc5",
    "config-opts": "",
    "sources": [
      {
        "type": "file",
        "url": "http://ftp.us.debian.org/debian/pool/main/v/vlc/vlc-nox_2.2.4-1~deb8u1_amd64.deb",
        "sha256": "56b0a9a2e3009515f1654a7fc960e528cbb78c054260c12addbd613fd13b9af9"
      }
    ]
  },
  {
    "name": "libzvbi-common",
    "config-opts": "",
    "sources": [
      {
        "type": "file",
        "url": "http://ftp.us.debian.org/debian/pool/main/z/zvbi/libzvbi0_0.2.35-3_amd64.deb",
        "sha256": "1a7b1c907235b3f75f85b263463daf0836a53e1dc432f45fed06e6c5e37e2369"
      }
    ]
  },
...
```

Duplicates are filtered out. If a package does not have an "original" tarball then the fields are left empty.

## Help

```
Usage of scraper:
  -arch string
      architecture of packages to download (default "amd64")
  -deb-version string
      debian code name to use (default "jessie")
  -mirror string
      mirror to use for downloading .deb packages (default "ftp.us.debian.org/debian")
  -pkg string
      debian package to be flatpaked
  -type string
      module type: valid types are 'deb' & 'tarball' (default "deb")
```
