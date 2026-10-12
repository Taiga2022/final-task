CREATE TABLE viewing_histories (
    user_id VARCHAR(64) NOT NULL,
    episode_id VARCHAR(64) NOT NULL,
    last_viewed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, episode_id),
    CONSTRAINT fk_viewing_histories_user FOREIGN KEY (user_id) REFERENCES `user` (id),
    CONSTRAINT fk_viewing_histories_episode FOREIGN KEY (episode_id) REFERENCES episode (id)
) ENGINE = InnoDB;
