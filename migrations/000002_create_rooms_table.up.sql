CREATE TABLE rooms (
    id VARCHAR(64) NOT NULL,
    room_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    owner_user_id VARCHAR(64) NOT NULL,
    weight DECIMAL(5, 4) NOT NULL DEFAULT 0.5,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT uq_rooms_room_name UNIQUE (room_name),
    CONSTRAINT chk_rooms_weight CHECK (weight >= 0 AND weight <= 1),
    CONSTRAINT fk_rooms_owner FOREIGN KEY (owner_user_id) REFERENCES `user` (id)
) ENGINE = InnoDB;
