
CREATE database 'tube';

-- tubeadmin
-- drowssap
CREATE USER 'tubeadmin'@'localhost' IDENTIFIED by 'drowssap';
GRANT ALL ON nontube.* TO 'tubeadmin'@'localhost';

CREATE TABLE 'users' (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `name` varchar(50) NOT NULL,
    `email` varchar(50) NOT NULL,
    `password` varchar(255) NOT NULL
);


CREATE TABLE 'videos' (
    `id` int NOT NULL PRIMARY KEY AUTO_INCREMENT,
    `uid` int,
    `title` varchar(255),
    `description` text,
    `duration` int,
    `likes` int,
    `dislikes` int,
);

