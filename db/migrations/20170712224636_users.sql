
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE users (
	id SERIAL,
	name VARCHAR(30),
	email VARCHAR(50) UNIQUE,
	hashed_password VARCHAR(64),
	salt VARCHAR(30),
	created_at DATETIME
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE users;

