
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	name VARCHAR(30) NOT NULL,
	email VARCHAR(50) NOT NULL UNIQUE,
	hashed_password VARCHAR(64) NOT NULL,
	salt VARCHAR(30) NOT NULL,
	created_at DATETIME NOT NULL
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE users;

