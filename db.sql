
CREATE database 'nontube';

CREATE USER 'tubeadmin'@'localhost' IDENTIFIED by 'drowssap';
GRANT ALL ON nontube.* TO 'tubeadmin'@'localhost';

CREATE TABLE `users` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` varchar(50) NOT NULL,
    `email` varchar(50) NOT NULL,
    `password` varchar(255) NOT NULL,
    `created_at` timestamp,
    `updated_at` timestamp,
    `deleted_at` timestamp
);

CREATE TABLE `videos` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `uid` int NOT NULL,
    `title` varchar(255) NOT NULL,
    `description` text,
    `url` varchar(511) NOT NULL,
    `thumbnail_url` varchar(511),
    `duration` int,
    `views` int,
    `likes` int,
    `dislikes` int,
    `created_at` timestamp,
    `updated_at` timestamp,
    `deleted_at` timestamp
);

CREATE TABLE `likes` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `uid` int NOT NULL,
    `v_id` int NOT NULL,
    `is_dislike` int NOT NULL DEFAULT 0
)
