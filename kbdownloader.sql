-- phpMyAdmin SQL Dump
-- version 4.7.1
-- https://www.phpmyadmin.net/
--
-- Host: mysql
-- Generation Time: 2018 年 7 月 08 日 15:15
-- サーバのバージョン： 5.7.18
-- PHP Version: 7.0.16

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET AUTOCOMMIT = 0;
START TRANSACTION;
SET time_zone = "+00:00";

--
-- Database: `kbdownloader`
--
CREATE DATABASE IF NOT EXISTS `kbdownloader` DEFAULT CHARACTER SET latin1 COLLATE latin1_swedish_ci;
USE `kbdownloader`;

-- --------------------------------------------------------

--
-- テーブルの構造 `package`
--

DROP TABLE IF EXISTS `package`;
CREATE TABLE IF NOT EXISTS `package` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `session_id` varchar(36) NOT NULL,
  `kbno` int(11) NOT NULL,
  `title` varchar(1024) DEFAULT NULL,
  `downloadLink` varchar(1024) DEFAULT NULL,
  `architecture` varchar(16) DEFAULT NULL,
  `fileName` varchar(1024) DEFAULT NULL,
  `language` varchar(16) DEFAULT NULL,
  `fileSize` int(11) DEFAULT NULL,
  `create_utc_date` datetime DEFAULT NULL,
  `update_utc_date` datetime DEFAULT NULL,
  `status` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_session_id` (`session_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

-- --------------------------------------------------------

--
-- テーブルの構造 `session`
--

DROP TABLE IF EXISTS `session`;
CREATE TABLE IF NOT EXISTS `session` (
  `id` varchar(36) NOT NULL,
  `kbno` int(11) NOT NULL,
  `saname` varchar(256) DEFAULT NULL,
  `sakey` varchar(256) DEFAULT NULL,
  `create_utc_date` datetime DEFAULT NULL,
  `update_utc_date` datetime DEFAULT NULL,
  `status` int(11) NOT NULL,
  PRIMARY KEY (`id`,`kbno`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

--
-- ダンプしたテーブルの制約
--

--
-- テーブルの制約 `package`
--
ALTER TABLE `package`
  ADD CONSTRAINT `fk_session_id` FOREIGN KEY (`session_id`) REFERENCES `session` (`id`);
COMMIT;

