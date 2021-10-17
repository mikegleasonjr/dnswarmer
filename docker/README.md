# dnswarmer

see https://github.com/mikegleasonjr/dnswarmer

## Version

0.0.1

## Architectures

* linux/arm/v7
* linux/arm64
* linux/amd64

## Usage

```
docker run -d \
    --name=dnswarmer \
    -p 53:5353/udp \
    --restart unless-stopped \
    mikegleasonjr/dnswarmer
```
