
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE followers (
	user_id BIGINT UNSIGNED,
	follower_id BIGINT UNSIGNED,
	created_at DATETIME,
	PRIMARY KEY(user_id, follower_id)
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE followers;
