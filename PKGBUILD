pkgname=wireguard-ddns
pkgver=1.0.0
pkgdesc="update endpoint addresses for WireGuard peers"
pkgrel=1
arch=('any')
makedepends=('go')
source=('go.mod'
        'go.sum'
        'main.go'
        'wireguard-ddns.service')
sha256sums=('18ae7b4f1f548ec8a6e75bf2832bbfe472b92f70856df1161ba0d87cd12e5ea5'
            'aa5acf0a7a4e81c206239d85d0bb81971524d59dd3ec2b833fd22a838d4ef4c1'
            '138ce5d0fe4afa8e0e4aafed697bd9688484b8e8ac8adbfda1d348f0cb266deb'
            '08cb15379fce390da242f2020ebfa33ec9f323e70d302fda885a247c320e2642')
OPTIONS=(strip !debug)

build() {
	cd "$srcdir"
	go mod download -x
	go build -trimpath -o "$srcdir/wireguard-ddns"
}

package() {
    mkdir -p "$pkgdir/usr/bin" "$pkgdir/usr/lib/systemd/system"
	cp "$srcdir/wireguard-ddns" "$pkgdir/usr/bin/wireguard-ddns"
	cp "$srcdir/wireguard-ddns.service" "$pkgdir/usr/lib/systemd/system/wireguard-ddns.service"
}