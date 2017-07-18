
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE posts (
	id SERIAL PRIMARY KEY,
	user_id BIGINT UNSIGNED NOT NULL,
	message VARCHAR(140) NOT NULL,
	create_at DATETIME NOT NULL
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE posts;
