-- Create users table
CREATE TABLE IF NOT EXISTS `users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `username` VARCHAR(64) NOT NULL,
    `password` VARCHAR(128) NOT NULL,
    `email` VARCHAR(128) DEFAULT NULL,
    `role` VARCHAR(32) NOT NULL DEFAULT 'user',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=active, 0=inactive',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`),
    UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create articles table
CREATE TABLE IF NOT EXISTS `articles` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `title` VARCHAR(256) NOT NULL,
    `content` TEXT,
    `author_id` BIGINT UNSIGNED NOT NULL,
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=published, 0=draft',
    `view_count` INT UNSIGNED NOT NULL DEFAULT 0,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_author_id` (`author_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert demo data
-- Password is 'password123' hashed with bcrypt (cost=10)
INSERT INTO `users` (`username`, `password`, `email`, `role`) VALUES
('admin', '$2a$10$7GCN6BtJO0OHv4n7/ec8fe5ijfzsyXhZmo92ldzJjo7uXqdTeUbLa', 'admin@example.com', 'admin'),
('user1', '$2a$10$7GCN6BtJO0OHv4n7/ec8fe5ijfzsyXhZmo92ldzJjo7uXqdTeUbLa', 'user1@example.com', 'user');

INSERT INTO `articles` (`title`, `content`, `author_id`, `status`) VALUES
('Welcome to Goupter', 'This is a demo article for the Goupter framework.', 1, 1),
('Getting Started', 'Learn how to build microservices with Goupter.', 1, 1);
