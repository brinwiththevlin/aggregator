cd sql/schema
goose postgres "postgres://brinhasavlin:postgres@localhost:5432/gator" down
goose postgres "postgres://brinhasavlin:postgres@localhost:5432/gator" up
cd ../..


