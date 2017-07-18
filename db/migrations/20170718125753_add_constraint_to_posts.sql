
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE posts ADD CONSTRAINT usersToPosts FOREIGN KEY(user_id) REFERENCES users(id);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE posts DROP FOREIGN KEY usersToPosts;
