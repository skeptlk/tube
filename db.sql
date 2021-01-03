
CREATE database 'nontube';

CREATE USER 'tubeadmin'@'localhost' IDENTIFIED by 'drowssap';
GRANT ALL ON nontube.* TO 'tubeadmin'@'localhost';

CREATE TABLE `users` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` varchar(50) NOT NULL,
    `email` varchar(50) NOT NULL,
    `password` varchar(255) NOT NULL,
    `is_admin` int NOT NULL DEFAULT 0,
    `created_at` timestamp,
    `updated_at` timestamp,
    `deleted_at` timestamp
);

CREATE TABLE `videos` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `user_id` int NOT NULL,
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

ALTER TABLE likes ADD CONSTRAINT fk_video_id FOREIGN KEY (v_id) REFERENCES videos(id) ON DELETE CASCADE;

CREATE TABLE `comments` (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `video_id` int NOT NULL,
    `user_id` int NOT NULL,
    `reply_to` int,
    `text` text NOT NULL,
    `created_at` timestamp,
    `updated_at` timestamp,
    `deleted_at` timestamp
);

ALTER TABLE comments ADD CONSTRAINT fk_comm_video_id  FOREIGN KEY (video_id) REFERENCES videos(id)   ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT fk_comm_user_id   FOREIGN KEY (user_id)  REFERENCES users(id)    ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT fk_comm_repl_comm FOREIGN KEY (reply_to) REFERENCES comments(id) ON DELETE CASCADE;




-- SELECT id, 
-- (SELECT SUM(views) FROM videos WHERE user_id = users.id) AS total_views, 
-- (SELECT COUNT(*) FROM videos WHERE user_id=users.id) AS num_videos 
-- FROM users WHERE deleted_at IS NULL;
