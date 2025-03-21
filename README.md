# Blobber

Blobber is a tool designed to detect and download publicly accessible blobs in Azure Blob Storage. This tool checks whether the Azure Storage accounts and containers you specify are publicly accessible, lists their contents, and downloads them if desired.

## Features

- Detect publicly accessible containers in Azure Storage accounts
- List blobs in discovered containers
- Batch download blobs
- Load account and container lists from files or specify individual values
- SSL verification support (can be skipped optionally)
- Configure the number of parallel requests and downloads
- Organize downloaded files in ACCOUNT/CONTAINER structure
- Monitor download and scan operations with visual progress bar
- Get detailed log information with debug mode
- Set limits for listing or downloading operations

## Installation

### Requirements

- Go 1.18 or higher

### Building from Source

```bash
git clone https://github.com/lodos2005/blobber.git
cd blobber
go build
```

### Download Pre-built Binary

You can download the latest version from the [GitHub Releases](https://github.com/lodos2005/blobber/releases) page.

## Usage

### Basic Parameters

```
  -a, --accounts string      Azure Storage account name or file containing account list
  -c, --containers string    Azure Storage container name or file containing container list
  -d, --download             Download found blobs
  -o, --output string        Output directory (for download) or file (for listing)
      --debug                Show detailed log information
      --skip-ssl             Skip SSL verification
  -p, --parallelism int      Number of parallel requests (default: 10)
  -l, --list                 List all found blob URLs
      --limit int            Limit the number of blobs to list or download (default: 10, not applied when writing to output file)
  -q, --quiet                Quiet mode, only show found containers
  -h, --help                 Show help information
```

### Usage Examples

#### Check Single Account and Container

```bash
./blobber -a mystorageaccount -c mycontainer
```

#### Check with Account and Container Lists

```bash
./blobber -a accounts.txt -c containers.txt
```

The account list (`accounts.txt`) and container list (`containers.txt`) files should contain one account or container name per line.

#### Download Found Blobs

```bash
./blobber -a mystorageaccount -c mycontainer --download
```

This command will download all found blobs to subdirectories with the `ACCOUNT/CONTAINER` structure.

#### Specify Output Directory

```bash
./blobber -a mystorageaccount -c mycontainer --download -o /path/to/output
```

This command will download the found blobs to the specified output directory with the `ACCOUNT/CONTAINER` structure.

#### Save Blob URL List to File

```bash
./blobber -a accounts.txt -c containers.txt -o blob-list.txt
```

This command saves all found blob URLs to the specified file.

#### Display Blob URL List

```bash
./blobber -a accounts.txt -c containers.txt --list
```

#### Set Limits

```bash
./blobber -a accounts.txt -c containers.txt --list --limit 5
```

This command displays at most 5 blob URLs to the screen.

```bash
./blobber -a mystorageaccount -c mycontainer --download --limit 10
```

This command downloads at most 10 blobs.

#### Run with Debug Mode

```bash
./blobber -a mystorageaccount -c mycontainer --debug
```

## How It Works

Blobber works as follows:

1. Reads the given account and container parameters
2. Checks the accessibility of accounts with DNS queries
3. Sends requests to the Azure Blob Storage API for each account and container combination
4. Gets the blob list for public containers
5. If the `--download` parameter is provided:
   - Downloads the found blobs
   - Saves them to the specified directory with `--output` or to the current directory in an `ACCOUNT/CONTAINER` structure
   - Skips files that already exist
6. If the `--list` parameter is provided, displays blob URLs on the screen
7. If the `--output` parameter is provided and the `--download` parameter is not, saves blob URLs to the specified file

## Limitations

- Downloading a large number of blobs with the `--download` parameter may require significant internet bandwidth and disk space.
- Operations may take a long time if there are many Azure Storage accounts.

## License

MIT License

## Contributing

1. Fork this project
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'New feature: feature description'`)
4. Push your branch (`git push origin my-new-feature`)
5. Open a Pull Request

---

# Blobber (Türkçe)

Blobber, Azure Blob Storage'da herkese açık (public) erişimli blobları tespit etmek ve indirmek için geliştirilmiş bir araçtır. Bu araç, belirttiğiniz Azure Storage hesapları ve container'ların public erişime açık olup olmadığını kontrol eder, açık olanların içeriğini listeler ve istenirse bu içerikleri indirir.

## Özellikler

- Azure Storage hesaplarında herkese açık container'ları tespit etme
- Bulunan container'lardaki blob'ları listeleme
- Blob'ları toplu olarak indirme
- Hesap ve container listelerini dosyadan yükleme veya tekil değer olarak belirtme
- SSL doğrulama desteği (isteğe bağlı olarak atlanabilir)
- Paralel istek ve indirme sayısını yapılandırma
- İndirilen dosyaları ACCOUNT/CONTAINER yapısında organize etme
- İndirme ve tarama işlemlerini görsel ilerleme çubuğu ile izleme
- Debug modu ile ayrıntılı log bilgisi alma
- Listele veya indirme işlemlerinde limit belirleme

## Kurulum

### Gereksinimler

- Go 1.18 veya üstü

### Kaynak Koddan Derleme

```bash
git clone https://github.com/lodos2005/blobber.git
cd blobber
go build
```

### Önceden Derlenmiş İkili Dosyayı İndirme

Son sürümü [GitHub Releases](https://github.com/lodos2005/blobber/releases) sayfasından indirebilirsiniz.

## Kullanım

### Temel Parametreler

```
  -a, --accounts string      Azure Storage hesabı adı veya hesap listesi içeren dosya
  -c, --containers string    Azure Storage container adı veya container listesi içeren dosya
  -d, --download             Bulunan blobları indir
  -o, --output string        Çıktı klasörü (indirme) veya dosyası (liste)
      --debug                Ayrıntılı log bilgisi göster
      --skip-ssl             SSL doğrulamasını atla
  -p, --parallelism int      Paralel istek sayısı (varsayılan: 10)
  -l, --list                 Bulunan tüm blobların URL'lerini ekrana yaz
      --limit int            Listeleme veya indirme işleminde gösterilecek/indirilecek maksimum blob sayısı (varsayılan: 10)
  -q, --quiet                Sessiz mod, sadece bulunan container'ları göster
  -h, --help                 Yardım bilgisini göster
```

### Kullanım Örnekleri

#### Tek Bir Hesap ve Container Kontrolü

```bash
./blobber -a mystorageaccount -c mycontainer
```

#### Hesap ve Container Listesi ile Kontrol

```bash
./blobber -a accounts.txt -c containers.txt
```

Hesap listesi (`accounts.txt`) ve container listesi (`containers.txt`) dosyaları, her satırda bir hesap veya container adı içermelidir.

#### Bulunan Blobları İndirme

```bash
./blobber -a mystorageaccount -c mycontainer --download
```

Bu komut, bulunan tüm blobları `ACCOUNT/CONTAINER` yapısında alt klasörlere indirecektir.

#### Çıktı Dizini Belirtme

```bash
./blobber -a mystorageaccount -c mycontainer --download -o /path/to/output
```

Bu komut, bulunan blob'ları belirtilen çıktı dizinine `ACCOUNT/CONTAINER` yapısında indirecektir.

#### Blob URL Listesini Dosyaya Kaydetme

```bash
./blobber -a accounts.txt -c containers.txt -o blob-list.txt
```

Bu komut, bulunan tüm blob URL'lerini belirtilen dosyaya kaydeder.

#### Blob URL Listesini Ekrana Yazdırma

```bash
./blobber -a accounts.txt -c containers.txt --list
```

#### Limit Belirleme

```bash
./blobber -a accounts.txt -c containers.txt --list --limit 5
```

Bu komut, en fazla 5 blob URL'sini ekrana yazdırır.

```bash
./blobber -a mystorageaccount -c mycontainer --download --limit 10
```

Bu komut, en fazla 10 blob'u indirir.

#### Debug Modu ile Çalıştırma

```bash
./blobber -a mystorageaccount -c mycontainer --debug
```

## Çalışma Mantığı

Blobber aşağıdaki şekilde çalışır:

1. Verilen hesap ve container parametrelerini okur
2. Hesapların erişilebilirliğini DNS sorguları ile kontrol eder
3. Her hesap ve container kombinasyonu için Azure Blob Storage API'sine istek gönderir
5. Public container'lar için blob listesini alır
6. Eğer `--download` parametresi verildiyse:
   - Bulunan blob'ları indirir
   - `--output` belirtildiyse o dizine, belirtilmediyse çalışılan dizine `ACCOUNT/CONTAINER` yapısında kaydeder
   - Aynı dosya zaten varsa indirmez
7. Eğer `--list` parametresi verildiyse, blob URL'lerini ekrana yazdırır
8. Eğer `--output` parametresi verilmiş ve `--download` parametresi verilmemişse, blob URL'lerini belirtilen dosyaya kaydeder

## Sınırlamalar

- `--download` parametresi kullanıldığında büyük miktarda blob indirmek, internet bağlantısı ve disk alanı gerektirebilir.
- Azure Storage hesaplarının çok sayıda olması durumunda işlemler uzun sürebilir.

## Lisans

MIT License

## Katkıda Bulunma

1. Bu projeyi fork edin
2. Özellik dalınızı oluşturun (`git checkout -b my-new-feature`)
3. Değişikliklerinizi commit edin (`git commit -am 'Yeni özellik: özellik açıklaması'`)
4. Dalınızı push edin (`git push origin my-new-feature`)
5. Bir Pull Request açın 