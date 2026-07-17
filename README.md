# Frontend Test — (Dashboard)

Dashboard artikel berbasis **Go + Gin + MySQL** untuk memenuhi
*Test Frontend*. Proyek ini **berdiri sendiri**
(terpisah dari `backend-test`) namun menggunakan pola yang sama:
memiliki migrasi database sendiri, koneksi MySQL sendiri, dan menyajikan
tampilan dashboard lewat server Go yang sama.

## Fitur (sesuai spec PDF)

1. **All Posts** — tab *Published*, *Drafts*, *Trashed*
   - Tiap tab menampilkan tabel berisi `title`, `category`, dan `action`
     (icon edit ✏️ & icon thrash 🗑️).
   - Klik edit → halaman edit (title, content, category bisa diedit) +
     tombol **Publish** & **Draft**.
   - Klik thrash → artikel dipindah ke tab *Trashed* (baris dipindah ke
     tabel `trash`).
2. **Add New** — form Title, Content, Category + tombol **Publish** & **Draft**.
3. **Preview** — menampilkan artikel bergaya blog dengan status `publish`,
   dilengkapi pagination.

## Struktur Folder

```
frontend-test/
├── cmd/
│   └── main.go                  # Entry point
├── database/
│   └── db.go                    # Koneksi MySQL & migrasi
├── models/
│   └── article.go               # Struct Article / ArticleInput
├── handlers/
│   └── article.go               # Logika CRUD + thrash + list
├── routes/
│   └── routes.go                # Pemetaan URL -> handler + serve dashboard
├── migrations/
│   ├── 001_create_articles_table.up.sql    # tabel articles
│   ├── 001_create_articles_table.down.sql
│   ├── 002_create_trash_table.up.sql       # tabel trash (kolom sama)
│   └── 002_create_trash_table.down.sql
├── public/
│   ├── index.html               # Dashboard (Tailwind via CDN)
│   └── app.js                   # Logika frontend (fetch API)
├── go.mod
├── .env
└── README.md
```

## Prasyarat

- **Go** 1.26.x — `go version`
- **MySQL** (XAMPP) menyala, dengan database `article` sudah dibuat:
  ```bash
  mysql -u root -e "CREATE DATABASE IF NOT EXISTS article;"
  ```
- File **`.env`** — salin dari `.env.example` dan isi sesuai setup MySQL
  kamu (lihat bagian Environment Variables). Tanpa `.env`, aplikasi pakai
  default bawaan (cocok untuk XAMPP lokal).

## Cara Install & Menjalankan

```bash
cd frontend-test
go mod tidy
cp .env.example .env      # buat file konfigurasi lokal (hanya sekali)
# edit .env sesuai setup MySQL kamu (lihat bagian Environment Variables)
go run ./cmd
```

Aplikasi akan:
1. Membaca konfigurasi dari file `.env` (fallback ke default bawaan
   jika `.env` tidak ada).
2. Menghubungkan ke MySQL.
3. Menjalankan migrasi otomatis → membuat tabel `articles` **dan** `trash`.
4. Menjalankan server di `http://localhost:8080`.

Buka **http://localhost:8080** di browser untuk melihat dashboard.

> **Penting:** Setiap ubah file `.go`, hentikan (`Ctrl+C`) lalu jalankan
> ulang `go run ./cmd`. Go mengkompilasi ke binary, jadi perubahan tidak
> otomatis masuk ke server yang sedang berjalan. Perubahan file `.env`
> juga butuh restart server agar dibaca ulang.

### File `.env`

Semua pengaturan koneksi database & server ada di file `.env` (bukan
lagi di-hardcode di `db.go`). Template-nya ada di `.env.example`:

```bash
# .env.example — salin ke .env lalu isi nilai kamu
DB_USER=root
DB_PASSWORD=
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=article
PORT=8080
RESET_DB=false
```

Aturan prioritas: **environment shell** > **nilai di `.env`** > **default
bawaan di kode** (jika `.env` tidak mengisi suatu key). File `.env` sudah
di-ignore oleh `.gitignore` agar rahasia tidak ke-commit.

### Reset Database (opsional)

Hapus tabel + migrasi dari awal (aman karena ini test):

```bash
RESET_DB=true go run ./cmd
```

## Endpoint API (internal dashboard)

| Method | URL                    | Keterangan                          |
|--------|------------------------|-------------------------------------|
| POST   | `/article/`            | Buat artikel (publish/draft)        |
| GET    | `/article/10/0`        | List articles (filter by tab)       |
| GET    | `/article/:id`         | Ambil satu artikel                  |
| PUT    | `/article/:id`         | Update artikel                      |
| POST   | `/article/:id/thrash`  | Pindahkan artikel ke tabel `trash`  |
| GET    | `/trash/10/0`          | List artikel di trash               |
| DELETE | `/article/:id`         | Hapus artikel (hard delete)         |

## Skema Database

Tabel `articles` dan `trash` memiliki kolom yang sama:

```sql
CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100) NOT NULL,
    created_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status ENUM('publish', 'draft', 'thrash') NOT NULL DEFAULT 'draft'
);

CREATE TABLE trash ( /* kolom sama dengan articles, status default 'thrash' */ );
```

Saat icon thrash diklik, baris dipindah dari `articles` ke `trash`
(menggunakan transaksi agar konsisten).

## Environment Variables

Semua variabel di bawah diisi di file **`.env`** (lihat bagian
[Cara Install & Menjalankan](#cara-install--menjalankan)). Nilai `Default`
baru dipakai jika key tersebut tidak diisi di `.env`.

| Variabel     | Default      | Fungsi                       |
|--------------|--------------|------------------------------|
| `DB_USER`    | `root`       | User MySQL                   |
| `DB_PASSWORD`| _(kosong)_   | Password MySQL               |
| `DB_HOST`    | `127.0.0.1`  | Host MySQL                   |
| `DB_PORT`    | `3306`       | Port MySQL                   |
| `DB_NAME`    | `article`    | Nama database                |
| `PORT`       | `8080`       | Port server dashboard        |
| `RESET_DB`   | _(off)_      | `true` untuk drop & migrasi ulang |

## Catatan

- Frontend menggunakan **Tailwind CSS via CDN** — tidak perlu build step
  untuk CSS.
- Proyek ini terpisah dari `backend-test` (dua repository berbeda), namun
  struktur kode mengikuti pola yang sama agar mudah dibaca.
