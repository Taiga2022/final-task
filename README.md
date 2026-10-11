

```sh
.
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── migrate/
│       ├── main.go
│       └── import_csv.go
├── internal/
│   ├── room/
│   │   ├── core/
│   │   │   ├── entity.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   └── adapter/
│   │       ├── http/
│   │       │   ├── handler.go
│   │       │   └── router.go
│   │       └── db/
│   │           └── repository.go
│   ├── reccomendation/
│   │   ├── core
│   │   └── adapter
│   ├── history/
│   │   ├── core
│   │   └── adapter
│   └── shared/
│       └── db/
│           └── connection.db
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── compose.yaml
├── mise.toml
├── go.mod
└── go.sum
```

