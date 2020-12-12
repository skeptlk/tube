
CREATE database 'tube';

-- tubeadmin
-- drowssap
CREATE USER 'tubeadmin'@'localhost' IDENTIFIED by 'drowssap';
GRANT ALL ON nontube.* TO 'tubeadmin'@'localhost';

CREATE table 'users' (
    `id` int(11) NOT NULL,
    `name` varchar(50) NOT NULL,
    `email` varchar(50) NOT NULL,
    `password` varchar(255) NOT NULL
);
