-- -------------------------------------------------------------
-- TablePlus 6.8.0(654)
--
-- https://tableplus.com/
--
-- Database: goeats_db
-- Generation Time: 2026-01-08 18:57:41.7460
-- -------------------------------------------------------------


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


DROP TABLE IF EXISTS `categories`;
CREATE TABLE `categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `category_name` varchar(100) NOT NULL,
  `description` text,
  `status` enum('active','inactive') DEFAULT 'active',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `customer`;
CREATE TABLE `customer` (
  `id` int NOT NULL AUTO_INCREMENT,
  `full_name` varchar(100) NOT NULL,
  `password` varchar(255) NOT NULL,
  `email` varchar(100) NOT NULL,
  `country_code` varchar(10) NOT NULL,
  `phone_number` varchar(20) NOT NULL,
  `dob` date DEFAULT NULL,
  `profile_image` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `login_id` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`),
  UNIQUE KEY `phone_number` (`phone_number`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `customer_delivery_addresses`;
CREATE TABLE `customer_delivery_addresses` (
  `id` int NOT NULL AUTO_INCREMENT,
  `customer_id` int NOT NULL,
  `full_name` varchar(100) DEFAULT NULL,
  `phone_number` varchar(15) DEFAULT NULL,
  `address` varchar(255) NOT NULL,
  `city` varchar(100) NOT NULL,
  `state` varchar(100) DEFAULT NULL,
  `country` varchar(100) DEFAULT NULL,
  `postal_code` varchar(15) DEFAULT NULL,
  `latitude` decimal(10,7) DEFAULT NULL,
  `longitude` decimal(10,7) DEFAULT NULL,
  `is_default` tinyint(1) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=20 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `delivery_partners`;
CREATE TABLE `delivery_partners` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `first_name` varchar(255) DEFAULT NULL,
  `last_name` varchar(255) DEFAULT NULL,
  `gender` enum('male','female') DEFAULT NULL,
  `date_of_birth` varchar(255) DEFAULT NULL,
  `primary_mobile` varchar(255) DEFAULT NULL,
  `blood_group` varchar(10) DEFAULT NULL,
  `city` varchar(255) DEFAULT NULL,
  `full_address` varchar(255) DEFAULT NULL,
  `languages_known` varchar(255) DEFAULT NULL,
  `profile_photo_url` text,
  `driving_license_url` text,
  `driving_license_number` varchar(50) DEFAULT NULL,
  `driving_license_expire` varchar(20) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `login_id` int DEFAULT NULL,
  `email` varchar(150) DEFAULT NULL,
  `status` enum('approved','rejected','pending') NOT NULL DEFAULT 'pending',
  `profile_completed` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `primary_mobile` (`primary_mobile`),
  UNIQUE KEY `email` (`email`),
  UNIQUE KEY `login_id` (`login_id`),
  CONSTRAINT `delivery_partners_ibfk_1` FOREIGN KEY (`login_id`) REFERENCES `login` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=57 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `login`;
CREATE TABLE `login` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `email` varchar(150) NOT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `type` varchar(50) DEFAULT NULL,
  `status` enum('active','inactive','blocked','pending') NOT NULL DEFAULT 'pending',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `email_verified` tinyint(1) DEFAULT '0',
  `verification_code` varchar(10) DEFAULT NULL,
  `password` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`),
  UNIQUE KEY `email_2` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=23 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `menu_items`;
CREATE TABLE `menu_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `category_id` bigint unsigned NOT NULL,
  `item_name` varchar(150) NOT NULL,
  `description` text,
  `price` decimal(10,2) NOT NULL,
  `image_url` varchar(255) DEFAULT NULL,
  `is_veg` tinyint(1) DEFAULT '0',
  `is_available` tinyint(1) DEFAULT '1',
  `preparation_time` int DEFAULT NULL,
  `status` enum('active','inactive') DEFAULT 'active',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_category` (`category_id`),
  CONSTRAINT `fk_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `restaurant_menu_items`;
CREATE TABLE `restaurant_menu_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `restaurant_id` bigint unsigned NOT NULL,
  `menu_item_id` bigint unsigned NOT NULL,
  `price` decimal(10,2) NOT NULL,
  `is_available` tinyint(1) DEFAULT '1',
  `preparation_time` int DEFAULT NULL,
  `status` enum('active','inactive') DEFAULT 'active',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_restaurant` (`restaurant_id`),
  KEY `fk_menu_item` (`menu_item_id`),
  CONSTRAINT `fk_menu_item` FOREIGN KEY (`menu_item_id`) REFERENCES `menu_items` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_restaurant` FOREIGN KEY (`restaurant_id`) REFERENCES `restaurants` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `restaurants`;
CREATE TABLE `restaurants` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `restaurant_name` varchar(150) NOT NULL,
  `business_owner_name` varchar(150) NOT NULL,
  `email` varchar(120) NOT NULL,
  `phone_number` varchar(30) NOT NULL,
  `password` varchar(255) NOT NULL,
  `business_address` text NOT NULL,
  `city` varchar(100) NOT NULL,
  `state` varchar(50) NOT NULL,
  `zipcode` varchar(20) NOT NULL,
  `latitude` decimal(10,7) DEFAULT NULL,
  `longitude` decimal(10,7) DEFAULT NULL,
  `business_description` text,
  `cover_image` varchar(255) DEFAULT NULL,
  `ein_number` varchar(20) DEFAULT NULL,
  `ssn_last4` char(4) DEFAULT NULL,
  `restaurant_permit_number` varchar(50) DEFAULT NULL,
  `bank_account_number` varchar(50) DEFAULT NULL,
  `routing_number` varchar(50) DEFAULT NULL,
  `status` enum('pending','approved','rejected','suspended') DEFAULT 'pending',
  `is_verified` tinyint(1) DEFAULT '0',
  `rating` float DEFAULT '0',
  `open_time` time DEFAULT NULL,
  `close_time` time DEFAULT NULL,
  `is_open` tinyint(1) DEFAULT '0',
  `minimum_order_amount` decimal(10,2) DEFAULT '0.00',
  `terms_accepted` tinyint(1) DEFAULT '0',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_contact_us`;
CREATE TABLE `tbl_contact_us` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_type` enum('customer','restaurant','delivery_partner','guest') NOT NULL DEFAULT 'guest',
  `user_id` bigint unsigned DEFAULT NULL,
  `name` varchar(100) NOT NULL,
  `email` varchar(150) NOT NULL,
  `phone` varchar(20) DEFAULT NULL,
  `message` text NOT NULL,
  `status` enum('new','in_progress','resolved') DEFAULT 'new',
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_customer_wishlist`;
CREATE TABLE `tbl_customer_wishlist` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `customer_id` int NOT NULL,
  `restaurant_id` bigint unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_customer_restaurants` (`customer_id`,`restaurant_id`),
  KEY `restaurant_id` (`restaurant_id`),
  CONSTRAINT `tbl_customer_wishlist_ibfk_1` FOREIGN KEY (`customer_id`) REFERENCES `customer` (`id`) ON DELETE CASCADE,
  CONSTRAINT `tbl_customer_wishlist_ibfk_2` FOREIGN KEY (`restaurant_id`) REFERENCES `restaurants` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_order_items`;
CREATE TABLE `tbl_order_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL,
  `menu_item_id` bigint unsigned NOT NULL,
  `title` varchar(255) DEFAULT NULL,
  `qty` int NOT NULL DEFAULT '1',
  `base_price` decimal(10,2) NOT NULL DEFAULT '0.00',
  `price` decimal(10,2) NOT NULL DEFAULT '0.00',
  `image_url` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `order_id` (`order_id`),
  CONSTRAINT `tbl_order_items_ibfk_1` FOREIGN KEY (`order_id`) REFERENCES `tbl_orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_orders`;
CREATE TABLE `tbl_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_number` varchar(50) NOT NULL,
  `customer_id` bigint unsigned NOT NULL,
  `restaurant_id` bigint unsigned NOT NULL,
  `address_id` bigint unsigned NOT NULL,
  `partner_id` bigint unsigned DEFAULT NULL,
  `subtotal` decimal(10,2) NOT NULL DEFAULT '0.00',
  `tax_amount` decimal(10,2) NOT NULL DEFAULT '0.00',
  `delivery_fee` decimal(10,2) NOT NULL DEFAULT '0.00',
  `tip_amount` decimal(10,2) DEFAULT '0.00',
  `total_amount` decimal(10,2) NOT NULL DEFAULT '0.00',
  `payment_method` enum('COD','Online') NOT NULL DEFAULT 'COD',
  `payment_status` enum('pending','success','failed') DEFAULT 'pending',
  `status` enum('pending','accepted','preparing','accepted_by_partner','pickup_ready','arrived_to_pickup','picked','on_the_way','delivered','cancelled') DEFAULT 'pending',
  `order_placed_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `delivery_time` datetime DEFAULT NULL,
  `cancel_reason` text,
  `rating` tinyint DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `delivery_otp` varchar(10) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_order_number` (`order_number`),
  KEY `idx_customer_id` (`customer_id`),
  KEY `idx_restaurant_id` (`restaurant_id`),
  KEY `idx_partner_id` (`partner_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=61 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_partner_bank_accounts`;
CREATE TABLE `tbl_partner_bank_accounts` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `partner_id` bigint NOT NULL,
  `stripe_account_id` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_payment_transactions`;
CREATE TABLE `tbl_payment_transactions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL,
  `customer_id` bigint unsigned NOT NULL,
  `transaction_reference` varchar(100) NOT NULL,
  `payment_mode` enum('COD','Online','Wallet','Card','UPI') NOT NULL DEFAULT 'COD',
  `payment_gateway` varchar(100) DEFAULT NULL,
  `amount` decimal(10,2) NOT NULL,
  `currency` varchar(10) DEFAULT 'INR',
  `status` enum('pending','success','failed','refunded') NOT NULL DEFAULT 'pending',
  `response_payload` text,
  `paid_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `provider` varchar(50) DEFAULT 'stripe',
  `brand` varchar(100) DEFAULT NULL,
  `last4` varchar(100) DEFAULT NULL,
  `payment_intent` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_order_id` (`order_id`),
  KEY `idx_customer_id` (`customer_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_transactions_order` FOREIGN KEY (`order_id`) REFERENCES `tbl_orders` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=39 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `tbl_ratings_reviews`;
CREATE TABLE `tbl_ratings_reviews` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `restaurant_id` bigint unsigned NOT NULL,
  `order_id` bigint unsigned DEFAULT NULL,
  `item_id` varchar(50) DEFAULT NULL,
  `rating` tinyint NOT NULL,
  `review` text,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  CONSTRAINT `tbl_ratings_reviews_chk_1` CHECK ((`rating` between 1 and 5))
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` int DEFAULT NULL,
  `name` text,
  `email` text,
  `phone` text,
  `address` text,
  `city` text,
  `state` text,
  `zipcode` int DEFAULT NULL,
  `country` text,
  `gender` text,
  `date_of_birth` text,
  `user_type` text,
  `profile_pic` text,
  `id_number` text,
  `id_doc_front` text,
  `id_doc_back` text,
  `is_active` int DEFAULT NULL,
  `created_at` text,
  `updated_at` text
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO `categories` (`id`, `category_name`, `description`, `status`, `created_at`, `updated_at`) VALUES
(1, 'Snacks', 'Quick bite and light food options', 'active', '2025-12-03 11:42:54', '2025-12-03 11:42:54'),
(2, 'Meal', 'Full meals including lunch and dinner items', 'active', '2025-12-03 11:42:54', '2025-12-03 11:42:54'),
(3, 'Dessert', 'Cakes, ice creams, sweets & bakery items', 'active', '2025-12-03 11:42:54', '2025-12-03 11:42:54'),
(4, 'Drinks', 'Cold drinks, shakes, coffee & beverages', 'active', '2025-12-03 11:42:54', '2025-12-03 11:42:54');

INSERT INTO `customer` (`id`, `full_name`, `password`, `email`, `country_code`, `phone_number`, `dob`, `profile_image`, `created_at`, `updated_at`, `login_id`) VALUES
(1, 'Atul Salunkhe', '$2a$10$xRY7ch0oGdFMKLm6A8/W1.7UlRiMRAn5yb0KZCDDuCEmqc.8h1TUa', 'atul.salunkhe+10@aviontechnology.us', '+91', '9876543217', '1996-05-04', '', '2025-11-29 19:03:22', '2025-12-06 15:47:40', 13),
(2, 'Atul Salunkhe', '$2a$10$y/XchebJTbsLsSTigTHAqOK31QIMEdT9s16RGhGfOh6zwQ7k6x4w.', 'atul.salunkhe+1@aviontechnology.us', '+91', '9876543222', '1998-03-20', 'https://example.com/pic.jpg', '2025-11-29 21:08:33', '2025-11-29 21:08:33', 14),
(3, 'Atul', '$2a$10$cAfvVejZrpYcA23GwM7Hd.YFv7MjNny5A0K8fLHIfeYv3.8O6fHy2', 'atu.salunkhe+2@aviontechnology.us', '1', '9876543210', '2004-11-29', '', '2025-11-29 21:57:35', '2025-11-29 21:57:35', 16),
(4, 'Test', '$2a$10$IWQfQrtLtAKrI5HIVZylCe/T2wLd/2Z9hZhwbczjlndQ.imwGIOaG', 'atul@gamil.com', '+1', '9087654321', '1996-11-29', '', '2025-11-29 21:59:27', '2025-11-29 21:59:27', 17),
(5, 'Danial m', '$2a$10$Qya8UUexQfaJCCf7fzAnGORANxJDpK6f.mnjrX0eXyFrPQILvgLPG', 'atul.salunkhe+7@aviontechnology.us', '+1', '97864523161', '1993-12-12', '', '2025-12-12 10:30:29', '2025-12-13 14:37:12', 18),
(6, 'Gorgeous m', '$2a$10$rK5jETRUJ4g/Btbzr/yyEO31VnXzysQyDzDsKw9Rl3HPlfJYWXo3W', 'atul.salunkhe+5@aviontechnology.us', '+1', '97856431284', '1983-12-18', '', '2025-12-18 15:01:07', '2025-12-18 18:01:55', 19),
(7, 'Jhon R', '$2a$10$WxNHUW/0TcK6tKhgur3wSOGKxI9ro.EeYXTt3LNYuImSg34sCmQ5u', 'atul.salunkhe+9@aviontechnology.us', '+91', '9876546789', '2002-12-20', '', '2025-12-20 12:39:15', '2025-12-20 12:44:13', 20),
(8, 'Martin R', '$2a$10$8ntnvoFeh.u5sbi0IxoJeuogRQO0wBfu43nfXMRKRHOL6f79.2IF.', 'atul.salunkhe+4@aviontechnology.us', '+1', '965464465614', '2001-01-07', '', '2026-01-07 10:47:56', '2026-01-07 10:47:56', 22);

INSERT INTO `customer_delivery_addresses` (`id`, `customer_id`, `full_name`, `phone_number`, `address`, `city`, `state`, `country`, `postal_code`, `latitude`, `longitude`, `is_default`, `created_at`, `updated_at`) VALUES
(11, 5, 'Danial m', '97864523161', 'Meghasparsh B Wing, B wing, Katraj - Dehu Road Bypass, Dhayari - 411046, MH, India', 'Dhayari', 'Maharashtra', 'India', '411046', 18.4430813, 73.8395638, 0, '2025-12-13 15:16:21', '2025-12-13 15:16:56'),
(12, 5, 'Danial m', '97864523161', 'Nandan PRO BIZ, Balewadi, Pune - 411045, MH, India', 'Pune', 'Maharashtra', 'India', '411045', 18.5727218, 73.7713188, 1, '2025-12-13 15:16:56', '2025-12-13 15:16:56'),
(14, 1, 'Atul Salunkhe', '9876543217', 'Trevega, University Road, Aundh, Pune - 411027, MH, India', 'Pune', 'Maharashtra', 'India', '411027', 18.5636746, 73.8135983, 0, '2025-12-16 10:01:31', '2025-12-16 10:09:31'),
(15, 1, 'Atul Salunkhe', '9876543217', 'Ramesh Dyeing, Sanghvi Kesari Road, Aundh, Pune - 411007, MH, India', 'Pune', 'Maharashtra', 'India', '411007', 18.5611379, 73.8137736, 1, '2025-12-16 10:09:31', '2025-12-20 11:03:08'),
(16, 6, 'Gorgeous m', '97856431284', 'Trevega, University Road, Aundh, Pune - 411027, MH, India', 'Pune', 'Maharashtra', 'India', '411027', 18.5636746, 73.8135983, 1, '2025-12-18 18:33:37', '2025-12-18 18:33:42'),
(18, 8, 'Martin R', '965464465614', 'Bird Valley, S.No.241/1, Datta Mandir Road, Wakad gaav, Pimpri - 411027, MH, India', 'Pimpri', 'Maharashtra', 'India', '411027', 18.5939789, 73.7640437, 1, '2026-01-07 10:51:01', '2026-01-07 10:51:06'),
(19, 7, 'Jhon R', '9876546789', 'Trevega, University Road, Aundh, Pune - 411027, MH, India', 'Pune', 'Maharashtra', 'India', '411027', 18.5636746, 73.8135983, 1, '2026-01-08 15:59:53', '2026-01-08 16:00:19');

INSERT INTO `delivery_partners` (`id`, `first_name`, `last_name`, `gender`, `date_of_birth`, `primary_mobile`, `blood_group`, `city`, `full_address`, `languages_known`, `profile_photo_url`, `driving_license_url`, `driving_license_number`, `driving_license_expire`, `created_at`, `updated_at`, `login_id`, `email`, `status`, `profile_completed`) VALUES
(1, 'kadir', 'pathan', 'male', '04/09/1997', '(897) 593-6675', NULL, NULL, NULL, NULL, 'http://localhost:8080/uploads/ids/1765270395220326000_18.jpg', 'https://f005.backblazeb2.com/file/GoEatspartner/1764306315612968000.jpg', 'MH1234434', '16/09/2029', '2025-11-24 14:46:22', '2025-12-09 14:23:15', 12, 'kadir.pathan@aviontechnology.us', 'approved', 1);

INSERT INTO `login` (`id`, `name`, `email`, `phone`, `type`, `status`, `created_at`, `updated_at`, `email_verified`, `verification_code`, `password`) VALUES
(8, 'John Doe', 'john@example.com', '+1 555-999-2222', 'customer', 'active', '2025-11-13 17:49:58', '2025-11-13 17:49:58', 0, NULL, ''),
(12, 'kadir pathan', 'kadir.pathan@aviontechnology.us', '(897) 593-6675', 'partner', 'active', '2025-11-24 14:46:22', '2025-11-27 11:29:31', 1, '583003', '$2a$10$sosd1cLlVQcv2THRGKjzA.iIciQXDNAsYPN4EQ0a1vcu5sqUMuf.O'),
(13, 'Atul Salunkhe', 'atul.salunkhe+10@aviontechnology.us', '9876543217', 'customer', 'active', '2025-11-29 19:03:22', '2025-12-18 14:58:05', 1, '183900', '$2a$10$xRY7ch0oGdFMKLm6A8/W1.7UlRiMRAn5yb0KZCDDuCEmqc.8h1TUa'),
(14, 'Atul Salunkhe', 'atul.salunkhe+1@aviontechnology.us', '9876543222', 'customer', 'inactive', '2025-11-29 21:08:33', '2025-11-29 21:08:33', 0, '', '$2a$10$y/XchebJTbsLsSTigTHAqOK31QIMEdT9s16RGhGfOh6zwQ7k6x4w.'),
(15, 'Atul', 'atu.salunkhe@aviontechnology.us', '9876543210', 'customer', 'inactive', '2025-11-29 21:39:23', '2025-11-29 21:39:23', 0, '', '$2a$10$qWigelPCaITmSuuhsOEl1.JbiGAioXl/KTCcWSYvTQwd7VKYoqTgS'),
(16, 'Atul', 'atu.salunkhe+2@aviontechnology.us', '9876543210', 'customer', 'inactive', '2025-11-29 21:57:35', '2025-11-29 21:57:35', 0, '', '$2a$10$cAfvVejZrpYcA23GwM7Hd.YFv7MjNny5A0K8fLHIfeYv3.8O6fHy2'),
(17, 'Test', 'atul@gamil.com', '9087654321', 'customer', 'blocked', '2025-11-29 21:59:27', '2025-12-26 18:11:56', 0, '', '$2a$10$IWQfQrtLtAKrI5HIVZylCe/T2wLd/2Z9hZhwbczjlndQ.imwGIOaG'),
(18, 'Danial m', 'atul.salunkhe+7@aviontechnology.us', '97864523161', 'customer', 'active', '2025-12-12 10:30:29', '2025-12-26 18:11:43', 1, '278602', '$2a$10$Qya8UUexQfaJCCf7fzAnGORANxJDpK6f.mnjrX0eXyFrPQILvgLPG'),
(19, 'Gorgeous m', 'atul.salunkhe+5@aviontechnology.us', '97856431284', 'customer', 'active', '2025-12-18 15:01:07', '2025-12-18 18:01:55', 1, '646797', '$2a$10$rK5jETRUJ4g/Btbzr/yyEO31VnXzysQyDzDsKw9Rl3HPlfJYWXo3W'),
(20, 'Jhon R', 'atul.salunkhe+9@aviontechnology.us', '9876546789', 'customer', 'active', '2025-12-20 12:39:15', '2026-01-08 18:12:03', 1, '788183', '$2a$10$WxNHUW/0TcK6tKhgur3wSOGKxI9ro.EeYXTt3LNYuImSg34sCmQ5u'),
(21, 'Kadir', 'admin@gmail.com', '9876543210', 'admin', 'active', '2025-12-23 14:48:42', '2025-12-29 11:52:08', 0, NULL, '$2a$10$cTvSkUz1GI9KFWBuO9ugOO2IaklBV7XGc7h3THMrfl0MulKuNm6S.'),
(22, 'Martin R', 'atul.salunkhe+4@aviontechnology.us', '965464465614', 'customer', 'active', '2026-01-07 10:47:56', '2026-01-07 10:50:32', 1, '608470', '$2a$10$8ntnvoFeh.u5sbi0IxoJeuogRQO0wBfu43nfXMRKRHOL6f79.2IF.');

INSERT INTO `menu_items` (`id`, `category_id`, `item_name`, `description`, `price`, `image_url`, `is_veg`, `is_available`, `preparation_time`, `status`, `created_at`, `updated_at`) VALUES
(1, 1, 'French Fries', 'Crispy golden fries', 5.99, 'https://example.com/images/fries.jpg', 1, 1, 10, 'active', '2025-12-03 11:48:22', '2025-12-03 11:48:22'),
(2, 1, 'Veg Cheese Sandwich', 'Grilled sandwich with fresh veggies', 6.99, 'https://example.com/paneer.jpg', 1, 1, 12, 'active', '2025-12-03 11:48:22', '2025-12-16 12:11:35'),
(3, 2, 'Chicken Burger Meal', 'Burger with fries & drink', 12.99, 'https://example.com/images/chickenmeal.jpg', 0, 1, 20, 'active', '2025-12-03 11:48:22', '2025-12-03 11:48:22'),
(4, 3, 'Chocolate Ice Cream', 'Creamy chocolate with toppings', 4.50, 'https://example.com/images/chocoicecream.jpg', 1, 1, 5, 'active', '2025-12-03 11:48:22', '2025-12-03 11:48:22'),
(5, 4, 'Cold Coffee', 'Iced coffee with cream', 3.99, 'https://example.com/images/coldcoffee.jpg', 1, 1, 7, 'active', '2025-12-03 11:48:22', '2025-12-03 11:48:22'),
(6, 2, 'Paneer Butter Masala', 'Rich creamy paneer gravy', 50.00, 'https://example.com/paneer.jpg', 1, 1, 18, 'active', '2025-12-16 11:30:13', '2025-12-16 12:21:04'),
(7, 2, 'Shahi Paneer', 'Rich and creamy shahi paneer', 60.00, 'https://example.com/shahi_paneer.jpg', 1, 0, 18, 'inactive', '2025-12-16 13:01:04', '2025-12-16 14:56:28'),
(8, 2, 'Fish Tikka', 'Originating from the coastal regions of India Fish Tikka is a flavorful and spicy dish thats a favorite among seafood enthusiasts This dish features succulent fish fillets marinated with a blend of turmeric yogurt lime juice and mustard seeds The marinade infuses the fish with a vibrant and tangy flavor while the spices add a delightful kick Grilled to perfection Fish Tikka delivers a deliciously smoky and aromatic experience that is both satisfying and unforgettable Whether enjoyed as an appetizer or a main course this dish is sure to delight anyone who appreciates the rich and diverse flavors of Indian cuisine', 5.66, 'https://cdn.shopify.com/s/files/1/0826/0553/9633/files/FishTikka_480x480.jpg?v=1705400135', 0, 1, 12, 'active', '2025-12-16 15:31:08', '2025-12-16 15:31:08');

INSERT INTO `restaurant_menu_items` (`id`, `restaurant_id`, `menu_item_id`, `price`, `is_available`, `preparation_time`, `status`, `created_at`, `updated_at`) VALUES
(1, 1, 1, 2.50, 1, 5, 'active', '2025-12-03 12:25:15', '2025-12-03 12:25:15'),
(2, 2, 1, 3.00, 1, 5, 'active', '2025-12-03 12:25:15', '2025-12-03 12:25:15'),
(3, 1, 2, 6.99, 1, 12, 'active', '2025-12-03 12:25:15', '2025-12-16 12:11:35'),
(4, 2, 3, 8.99, 1, 10, 'active', '2025-12-03 12:25:15', '2025-12-03 12:25:15'),
(5, 1, 6, 50.00, 1, 18, 'active', '2025-12-16 11:30:13', '2025-12-16 12:21:04'),
(6, 5, 6, 50.00, 1, 15, 'active', '2025-12-16 12:46:50', '2025-12-16 12:46:50'),
(7, 5, 7, 60.00, 0, 18, 'inactive', '2025-12-16 13:01:04', '2025-12-16 14:56:28'),
(8, 5, 8, 5.66, 1, 12, 'active', '2025-12-16 15:31:08', '2025-12-16 15:31:08');

INSERT INTO `restaurants` (`id`, `restaurant_name`, `business_owner_name`, `email`, `phone_number`, `password`, `business_address`, `city`, `state`, `zipcode`, `latitude`, `longitude`, `business_description`, `cover_image`, `ein_number`, `ssn_last4`, `restaurant_permit_number`, `bank_account_number`, `routing_number`, `status`, `is_verified`, `rating`, `open_time`, `close_time`, `is_open`, `minimum_order_amount`, `terms_accepted`, `created_at`, `updated_at`) VALUES
(1, 'Tasty Bites', 'John Smith', 'contact@tastybites.com', '+1 323-555-7890', '$2y$10$abcdefghijklmnopqrstuvhashedpasswordexample', '123 Main Street', 'Los Angeles', 'CA', '90001', 34.0522350, -118.2436830, 'Mexican & American fusion fast food restaurant', 'https://example.com/cover/tastybites.jpg', '12-3456789', '1234', 'LA-987654', '123456789', '021000021', 'approved', 1, 4.5, '10:00:00', '22:00:00', 1, 20.00, 1, '2025-12-03 11:37:32', '2025-12-03 11:37:32'),
(2, 'Pizza House', 'Mike Johnson', 'owner@pizzahouse.com', '+1 702-555-1234', '$2y$10$hashhereexample', '456 Downtown Road', 'Las Vegas', 'NV', '88901', 36.1716000, -115.1391000, 'Fresh wood-fired Italian style pizzas', 'https://example.com/cover/pizzahouse.jpg', '98-7654321', '5678', 'NV-123789', '987654321', '122105155', 'pending', 0, 0, '11:00:00', '23:30:00', 1, 15.00, 1, '2025-12-03 11:37:32', '2025-12-24 11:45:37'),
(3, 'Burger Hub', 'David Lee', 'info@burgerhub.com', '+1 415-555-9988', '$2y$10$32dddsasdlinpwdexample', '85 Market Street', 'San Francisco', 'CA', '94105', 37.7749000, -122.4194000, 'Premium gourmet burgers and shakes', 'https://example.com/cover/burgerhub.jpg', '54-9876543', '9876', 'CA-323444', '1122334455', '121000358', 'approved', 1, 4.3, '09:30:00', '21:00:00', 1, 18.50, 1, '2025-12-03 11:37:32', '2025-12-03 11:37:32'),
(4, 'Healthy Bowl', 'Emily Davis', 'support@healthybowl.com', '+1 646-555-4432', '$2y$10$isjsnslw998877example', '734 5th Avenue', 'New York', 'NY', '10019', 40.7128000, -74.0060000, 'Organic salads, bowls & smoothies', 'https://example.com/cover/healthybowl.jpg', '22-4443322', '1122', 'NY-876555', '5566778899', '026009593', 'approved', 1, 4.8, '08:00:00', '20:00:00', 1, 10.00, 1, '2025-12-03 11:37:32', '2025-12-03 11:37:32'),
(5, 'Medieval Times Dinner & Tournament - Buena Park', 'kadir  pathan', 'kadir@gmail.com', '(897) 593-6675', '$2a$10$Ox22C8IxwwdyeJiEF7nOOujJhThJ6WiCY9Do/f2h/rptX/anmWIKG', '123 FC Road, Shivaji Nagar', 'Pune', 'Maharashtra', '411005', 18.5630420, 73.8136030, NULL, NULL, NULL, NULL, NULL, 'acct_1SdAPDCt6k2m5jtW', NULL, 'approved', 0, 0, '10:00:00', '16:00:00', 1, 0.00, 0, '2025-12-11 16:50:16', '2026-01-06 18:37:48'),
(6, 'Medieval Times Dinner & Tournament - Buena Park', 'kadir  pathan', 'kadir+02@gmail.com', '(897) 593-6675', '$2a$10$94GePBdSOxHH46LJuv0XpeBI70h5YmmpxQE0cIeR.VYA1rqwLMbva', '7662 Beach Boulevard, Buena Park, CA 90620, United States of America', 'Buena Park', 'California', '90620', 33.8513540, -117.9973130, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'approved', 0, 0, NULL, NULL, 0, 0.00, 0, '2025-12-11 16:56:51', '2025-12-11 18:19:48');

INSERT INTO `tbl_contact_us` (`id`, `user_type`, `user_id`, `name`, `email`, `phone`, `message`, `status`, `created_at`, `updated_at`) VALUES
(1, 'customer', 101, 'Jack', 'atul@gmail.com', '9876543210', 'Payment deducted but order not placed', 'new', '2025-12-13 14:54:31', '2025-12-13 14:54:31'),
(2, 'customer', 5, 'Test', 'Test@gmail.com', '97864523161', 'payment deducted but order not placed', 'new', '2025-12-13 15:06:50', '2025-12-13 15:06:50');

INSERT INTO `tbl_customer_wishlist` (`id`, `customer_id`, `restaurant_id`, `created_at`) VALUES
(9, 1, 1, '2025-12-15 10:03:18'),
(12, 1, 5, '2025-12-31 14:25:02');

INSERT INTO `tbl_order_items` (`id`, `order_id`, `menu_item_id`, `title`, `qty`, `base_price`, `price`, `image_url`, `created_at`, `updated_at`) VALUES
(1, 21, 1, NULL, 9, 53.91, 53.91, NULL, '2025-12-09 19:02:46', '2025-12-09 19:02:46'),
(2, 21, 2, NULL, 2, 12.98, 12.98, NULL, '2025-12-09 19:02:46', '2025-12-09 19:02:46'),
(3, 22, 3, NULL, 2, 25.98, 25.98, NULL, '2025-12-09 19:18:06', '2025-12-09 19:18:06'),
(4, 23, 3, NULL, 2, 25.98, 25.98, NULL, '2025-12-10 12:42:45', '2025-12-10 12:42:45'),
(5, 24, 2, '', 2, 12.98, 12.98, NULL, '2025-12-10 12:47:29', '2025-12-10 12:47:29'),
(6, 25, 1, 'French Fries', 2, 11.98, 11.98, NULL, '2025-12-10 12:49:34', '2025-12-10 12:49:34'),
(7, 26, 3, 'Chicken Burger Meal', 10, 129.90, 129.90, NULL, '2025-12-10 16:24:36', '2025-12-10 16:24:36'),
(8, 27, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-10 16:27:39', '2025-12-10 16:27:39'),
(9, 27, 2, 'Veg Cheese Sandwich', 1, 6.49, 6.49, NULL, '2025-12-10 16:27:39', '2025-12-10 16:27:39'),
(10, 28, 3, 'Chicken Burger Meal', 4, 51.96, 51.96, NULL, '2025-12-12 11:10:09', '2025-12-12 11:10:09'),
(15, 31, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:14:50', '2025-12-12 17:14:50'),
(16, 32, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:15:40', '2025-12-12 17:15:40'),
(17, 33, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:27:48', '2025-12-12 17:27:48'),
(18, 34, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:42:30', '2025-12-12 17:42:30'),
(19, 35, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:45:58', '2025-12-12 17:45:58'),
(20, 36, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-12 17:46:45', '2025-12-12 17:46:45'),
(22, 38, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-12 19:41:19', '2025-12-12 19:41:19'),
(23, 39, 3, 'Chicken Burger Meal', 2, 25.98, 25.98, NULL, '2025-12-13 15:18:05', '2025-12-13 15:18:05'),
(24, 40, 1, 'French Fries', 3, 17.97, 17.97, NULL, '2025-12-13 15:19:56', '2025-12-13 15:19:56'),
(25, 40, 2, 'Veg Cheese Sandwich', 2, 12.98, 12.98, NULL, '2025-12-13 15:19:56', '2025-12-13 15:19:56'),
(26, 41, 3, 'Chicken Burger Meal', 1, 12.99, 12.99, NULL, '2025-12-18 18:17:08', '2025-12-18 18:17:08'),
(27, 42, 3, 'Chicken Burger Meal', 2, 25.98, 25.98, NULL, '2025-12-18 18:34:51', '2025-12-18 18:34:51'),
(28, 43, 6, 'Paneer Butter Masala', 2, 100.00, 100.00, NULL, '2025-12-19 17:51:37', '2025-12-19 17:51:37'),
(29, 44, 3, 'Chicken Burger Meal', 4, 51.96, 51.96, NULL, '2025-12-19 18:23:51', '2025-12-19 18:23:51'),
(30, 45, 3, 'Chicken Burger Meal', 2, 25.98, 25.98, NULL, '2025-12-20 11:06:30', '2025-12-20 11:06:30'),
(31, 46, 3, 'Chicken Burger Meal', 2, 25.98, 25.98, NULL, '2025-12-20 11:22:32', '2025-12-20 11:22:32'),
(32, 47, 1, 'French Fries', 3, 17.97, 17.97, NULL, '2025-12-20 11:45:00', '2025-12-20 11:45:00'),
(33, 47, 2, 'Veg Cheese Sandwich', 5, 34.95, 34.95, NULL, '2025-12-20 11:45:00', '2025-12-20 11:45:00'),
(34, 48, 6, 'Paneer Butter Masala', 2, 100.00, 100.00, NULL, '2025-12-20 11:56:42', '2025-12-20 11:56:42'),
(35, 49, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-20 11:58:44', '2025-12-20 11:58:44'),
(36, 49, 2, 'Veg Cheese Sandwich', 3, 20.97, 20.97, NULL, '2025-12-20 11:58:44', '2025-12-20 11:58:44'),
(37, 50, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-20 12:05:06', '2025-12-20 12:05:06'),
(38, 50, 2, 'Veg Cheese Sandwich', 1, 6.99, 6.99, NULL, '2025-12-20 12:05:06', '2025-12-20 12:05:06'),
(39, 51, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-20 12:10:07', '2025-12-20 12:10:07'),
(40, 51, 2, 'Veg Cheese Sandwich', 1, 6.99, 6.99, NULL, '2025-12-20 12:10:07', '2025-12-20 12:10:07'),
(41, 52, 1, 'French Fries', 1, 5.99, 5.99, NULL, '2025-12-20 12:30:20', '2025-12-20 12:30:20'),
(42, 52, 6, 'Paneer Butter Masala', 1, 50.00, 50.00, NULL, '2025-12-20 12:30:20', '2025-12-20 12:30:20'),
(43, 53, 1, 'French Fries', 5, 29.95, 29.95, NULL, '2025-12-20 14:02:29', '2025-12-20 14:02:29'),
(44, 54, 1, 'French Fries', 2, 11.98, 11.98, NULL, '2025-12-29 14:27:03', '2025-12-29 14:27:03'),
(45, 55, 2, 'Veg Cheese Sandwich', 2, 13.98, 13.98, NULL, '2025-12-29 14:27:47', '2025-12-29 14:27:47'),
(46, 56, 8, 'Fish Tikka', 3, 16.98, 16.98, NULL, '2026-01-06 17:00:45', '2026-01-06 17:00:45'),
(47, 57, 8, 'Fish Tikka', 2, 11.32, 11.32, 'https://cdn.shopify.com/s/files/1/0826/0553/9633/files/FishTikka_480x480.jpg?v=1705400135', '2026-01-06 17:45:20', '2026-01-06 17:45:20'),
(48, 58, 8, 'Fish Tikka', 1, 5.66, 5.66, 'https://cdn.shopify.com/s/files/1/0826/0553/9633/files/FishTikka_480x480.jpg?v=1705400135', '2026-01-07 10:52:09', '2026-01-07 10:52:09'),
(49, 59, 6, 'Paneer Butter Masala', 1, 50.00, 50.00, 'https://example.com/paneer.jpg', '2026-01-08 14:02:30', '2026-01-08 14:02:30'),
(50, 60, 8, 'Fish Tikka', 1, 5.66, 5.66, 'https://cdn.shopify.com/s/files/1/0826/0553/9633/files/FishTikka_480x480.jpg?v=1705400135', '2026-01-08 14:11:08', '2026-01-08 14:11:08');

INSERT INTO `tbl_orders` (`id`, `order_number`, `customer_id`, `restaurant_id`, `address_id`, `partner_id`, `subtotal`, `tax_amount`, `delivery_fee`, `tip_amount`, `total_amount`, `payment_method`, `payment_status`, `status`, `order_placed_at`, `delivery_time`, `cancel_reason`, `rating`, `created_at`, `updated_at`, `delivery_otp`) VALUES
(21, '#GOEATS-20251209-00001', 1, 1, 8, NULL, 66.89, 5.00, 3.00, 0.00, 74.89, 'Online', 'pending', 'cancelled', '2025-12-09 19:02:46', NULL, 'Incorrect item selected', NULL, '2025-12-09 19:02:46', '2025-12-20 11:08:02', NULL),
(22, '#GOEATS-20251209-00022', 1, 2, 8, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'pending', 'cancelled', '2025-12-09 19:18:06', NULL, 'Incorrect item selected', NULL, '2025-12-09 19:18:06', '2025-12-11 14:28:26', NULL),
(23, '#GOEATS-20251210-00023', 1, 2, 8, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'success', 'preparing', '2025-12-10 12:42:45', NULL, NULL, NULL, '2025-12-10 12:42:45', '2025-12-10 17:08:01', NULL),
(24, '#GOEATS-20251210-00024', 1, 1, 8, NULL, 12.98, 5.00, 3.00, 0.00, 20.98, 'Online', 'success', 'pickup_ready', '2025-12-10 12:47:29', NULL, NULL, NULL, '2025-12-10 12:47:29', '2025-12-10 17:08:01', NULL),
(25, '#GOEATS-20251210-00025', 1, 1, 8, NULL, 11.98, 5.00, 3.00, 0.00, 19.98, 'Online', 'success', 'picked', '2025-12-10 12:49:34', NULL, NULL, NULL, '2025-12-10 12:49:34', '2025-12-10 17:08:01', NULL),
(26, '#GOEATS-20251210-00026', 1, 2, 7, NULL, 129.90, 5.00, 3.00, 0.00, 137.90, 'Online', 'success', 'on_the_way', '2025-12-10 16:24:36', NULL, NULL, NULL, '2025-12-10 16:24:36', '2025-12-10 17:08:01', NULL),
(27, '#GOEATS-20251210-00027', 1, 1, 7, NULL, 12.48, 5.00, 3.00, 0.00, 20.48, 'Online', 'success', 'delivered', '2025-12-10 16:27:39', NULL, NULL, NULL, '2025-12-10 16:27:39', '2025-12-10 17:08:01', NULL),
(28, '#GOEATS-20251212-00028', 1, 2, 6, NULL, 51.96, 5.00, 3.00, 0.00, 59.96, 'Online', 'success', 'pending', '2025-12-12 11:10:09', NULL, NULL, NULL, '2025-12-12 11:10:09', '2025-12-12 11:10:10', NULL),
(31, '#GOEATS-20251212-00029', 1, 2, 6, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-12 17:14:50', NULL, NULL, NULL, '2025-12-12 17:14:50', '2025-12-12 17:14:50', NULL),
(32, '#GOEATS-20251212-00032', 1, 2, 6, 21, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'delivered', '2025-12-12 17:15:40', NULL, NULL, NULL, '2025-12-12 17:15:40', '2026-01-06 10:33:08', NULL),
(33, '#GOEATS-20251212-00033', 1, 2, 6, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-12 17:27:48', NULL, NULL, NULL, '2025-12-12 17:27:48', '2025-12-12 17:27:49', NULL),
(34, '#GOEATS-20251212-00034', 6, 2, 11, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-12 17:42:30', NULL, NULL, NULL, '2025-12-12 17:42:30', '2025-12-12 17:42:31', NULL),
(35, '#GOEATS-20251212-00035', 6, 2, 11, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-12 17:45:58', NULL, NULL, NULL, '2025-12-12 17:45:58', '2025-12-12 17:45:59', NULL),
(36, '#GOEATS-20251212-00036', 6, 2, 11, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-12 17:46:45', NULL, NULL, NULL, '2025-12-12 17:46:45', '2025-12-12 17:46:45', NULL),
(38, '#GOEATS-20251212-00037', 1, 1, 6, NULL, 5.99, 5.00, 3.00, 0.00, 13.99, 'Online', 'success', 'cancelled', '2025-12-12 19:41:19', NULL, 'Incorrect item selected', NULL, '2025-12-12 19:41:19', '2025-12-20 11:34:58', NULL),
(39, '#GOEATS-20251213-00039', 5, 2, 12, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'success', 'cancelled', '2025-12-13 15:18:05', NULL, 'Changed my mind', NULL, '2025-12-13 15:18:05', '2025-12-13 15:18:21', NULL),
(40, '#GOEATS-20251213-00040', 5, 1, 12, NULL, 30.95, 5.00, 3.00, 0.00, 38.95, 'Online', 'success', 'pending', '2025-12-13 15:19:56', NULL, NULL, NULL, '2025-12-13 15:19:56', '2025-12-13 15:19:57', NULL),
(41, '#GOEATS-20251218-00041', 6, 2, 11, NULL, 12.99, 5.00, 3.00, 0.00, 20.99, 'Online', 'success', 'pending', '2025-12-18 18:17:08', NULL, NULL, NULL, '2025-12-18 18:17:08', '2025-12-18 18:17:08', NULL),
(42, '#GOEATS-20251218-00042', 6, 2, 16, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'success', 'pending', '2025-12-18 18:34:51', NULL, NULL, NULL, '2025-12-18 18:34:51', '2025-12-18 18:34:51', NULL),
(43, '#GOEATS-20251219-00043', 6, 1, 16, NULL, 100.00, 5.00, 3.00, 0.00, 108.00, 'Online', 'success', 'pending', '2025-12-19 17:51:37', NULL, NULL, NULL, '2025-12-19 17:51:37', '2025-12-19 17:51:38', NULL),
(44, '#GOEATS-20251219-00044', 6, 2, 16, NULL, 51.96, 5.00, 3.00, 0.00, 59.96, 'Online', 'success', 'cancelled', '2025-12-19 18:23:51', NULL, 'Order placed by mistake', NULL, '2025-12-19 18:23:51', '2025-12-19 18:24:48', NULL),
(45, '#GOEATS-20251220-00045', 1, 2, 15, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'success', 'cancelled', '2025-12-20 11:06:30', NULL, 'Found a better option', NULL, '2025-12-20 11:06:30', '2025-12-20 12:24:52', NULL),
(46, '#GOEATS-20251220-00046', 1, 2, 15, NULL, 25.98, 5.00, 3.00, 0.00, 33.98, 'Online', 'success', 'cancelled', '2025-12-20 11:22:32', NULL, 'Order placed by mistake', NULL, '2025-12-20 11:22:32', '2025-12-20 11:24:13', NULL),
(47, '#GOEATS-20251220-00047', 1, 1, 15, NULL, 52.92, 5.00, 3.00, 0.00, 60.92, 'Online', 'success', 'cancelled', '2025-12-20 11:45:00', NULL, 'Changed my mind', NULL, '2025-12-20 11:45:00', '2025-12-20 12:17:40', NULL),
(48, '#GOEATS-20251220-00048', 1, 1, 15, 21, 100.00, 5.00, 3.00, 0.00, 108.00, 'Online', 'success', 'delivered', '2025-12-20 11:56:42', NULL, NULL, NULL, '2025-12-20 11:56:42', '2026-01-06 10:30:52', NULL),
(49, '#GOEATS-20251220-00049', 1, 1, 15, NULL, 26.96, 5.00, 3.00, 0.00, 34.96, 'Online', 'success', 'pending', '2025-12-20 11:58:44', NULL, NULL, NULL, '2025-12-20 11:58:44', '2025-12-20 11:58:45', NULL),
(50, '#GOEATS-20251220-00050', 1, 1, 15, NULL, 12.98, 5.00, 3.00, 0.00, 20.98, 'Online', 'success', 'pending', '2025-12-20 12:05:06', NULL, NULL, NULL, '2025-12-20 12:05:06', '2025-12-20 12:05:06', NULL),
(51, '#GOEATS-20251220-00051', 1, 1, 15, NULL, 12.98, 5.00, 3.00, 0.00, 20.98, 'Online', 'success', 'pending', '2025-12-20 12:10:07', NULL, NULL, NULL, '2025-12-20 12:10:07', '2025-12-20 12:10:07', NULL),
(52, '#GOEATS-20251220-00052', 1, 1, 15, NULL, 55.99, 5.00, 3.00, 0.00, 63.99, 'Online', 'success', 'delivered', '2025-12-20 12:30:20', NULL, NULL, NULL, '2025-12-20 12:30:20', '2025-12-23 15:46:09', NULL),
(53, '#GOEATS-20251220-00053', 7, 1, 17, NULL, 29.95, 5.00, 3.00, 0.00, 37.95, 'Online', 'success', 'cancelled', '2025-12-20 14:02:29', NULL, 'Incorrect item selected', NULL, '2025-12-20 14:02:29', '2025-12-20 14:16:45', NULL),
(54, '#GOEATS-20251229-00054', 7, 1, 17, 21, 11.98, 5.00, 3.00, 0.00, 19.98, 'Online', 'success', 'delivered', '2025-12-29 14:27:03', NULL, NULL, NULL, '2025-12-29 14:27:03', '2026-01-06 10:27:36', NULL),
(55, '#GOEATS-20251229-00055', 7, 1, 17, 21, 13.98, 5.00, 3.00, 0.00, 21.98, 'Online', 'success', 'delivered', '2025-12-29 14:27:47', NULL, NULL, NULL, '2025-12-29 14:27:47', '2026-01-06 10:25:52', NULL),
(56, '#GOEATS-20260106-00056', 1, 5, 15, NULL, 16.98, 5.00, 3.00, 0.00, 24.98, 'Online', 'success', 'pending', '2026-01-06 17:00:45', NULL, NULL, NULL, '2026-01-06 17:00:45', '2026-01-06 17:00:46', NULL),
(57, '#GOEATS-20260106-00057', 1, 5, 15, NULL, 11.32, 5.00, 3.00, 0.00, 19.32, 'Online', 'success', 'pending', '2026-01-06 17:45:20', NULL, NULL, NULL, '2026-01-06 17:45:20', '2026-01-06 17:45:21', NULL),
(58, '#GOEATS-20260107-00058', 8, 5, 18, NULL, 5.66, 5.00, 3.00, 0.00, 13.66, 'Online', 'success', 'cancelled', '2026-01-07 10:52:09', NULL, 'Changed my mind', NULL, '2026-01-07 10:52:09', '2026-01-07 10:54:08', NULL),
(59, '#GOEATS-20260108-00059', 8, 1, 18, NULL, 50.00, 5.00, 3.00, 0.00, 58.00, 'Online', 'success', 'pending', '2026-01-08 14:02:30', NULL, NULL, NULL, '2026-01-08 14:02:30', '2026-01-08 14:02:30', NULL),
(60, '#GOEATS-20260108-00060', 8, 5, 18, NULL, 5.66, 5.00, 3.00, 0.00, 13.66, 'Online', 'success', 'pending', '2026-01-08 14:11:08', NULL, NULL, NULL, '2026-01-08 14:11:08', '2026-01-08 14:11:08', NULL);

INSERT INTO `tbl_partner_bank_accounts` (`id`, `partner_id`, `stripe_account_id`, `created_at`, `updated_at`) VALUES
(2, 17, 'acct_1SaWWHERnkN7VCPg', '2025-12-04 12:30:07', '2025-12-04 12:30:07'),
(7, 12, 'acct_1SaaRmCYWiQHdtpi', '2025-12-04 16:41:45', '2025-12-04 16:41:45');

INSERT INTO `tbl_payment_transactions` (`id`, `order_id`, `customer_id`, `transaction_reference`, `payment_mode`, `payment_gateway`, `amount`, `currency`, `status`, `response_payload`, `paid_at`, `created_at`, `updated_at`, `provider`, `brand`, `last4`, `payment_intent`) VALUES
(1, 21, 1, '', 'Card', 'stripe', 74.89, 'INR', 'refunded', NULL, NULL, '2025-12-09 19:02:47', '2025-12-20 11:08:02', 'stripe', 'visa', '1111', 'pi_3ScR1dCY24ISQw4B01qhhJOn'),
(2, 22, 1, '', 'Card', 'stripe', 33.98, 'INR', 'success', NULL, NULL, '2025-12-09 19:18:06', '2025-12-09 19:18:06', 'stripe', 'visa', '1111', 'pi_3ScRGVCY24ISQw4B1ZV6hEoH'),
(3, 23, 1, '', 'Card', 'stripe', 33.98, 'INR', 'success', NULL, NULL, '2025-12-10 12:42:46', '2025-12-10 12:42:46', 'stripe', 'visa', '1111', 'pi_3SchZZCY24ISQw4B0eGJCF9x'),
(4, 24, 1, '', 'Card', 'stripe', 20.98, 'INR', 'success', NULL, NULL, '2025-12-10 12:47:29', '2025-12-10 12:47:29', 'stripe', 'visa', '1111', 'pi_3Sche6CY24ISQw4B1HRi2SzN'),
(5, 25, 1, '', 'Card', 'stripe', 19.98, 'INR', 'success', NULL, NULL, '2025-12-10 12:49:34', '2025-12-10 12:49:34', 'stripe', 'visa', '1111', 'pi_3Schg8CY24ISQw4B1ySZVVSr'),
(6, 26, 1, '', 'Card', 'stripe', 137.90, 'INR', 'success', NULL, NULL, '2025-12-10 16:24:37', '2025-12-10 16:24:37', 'stripe', 'visa', '1111', 'pi_3Scl22CY24ISQw4B11eb8pFc'),
(7, 27, 1, '', 'Card', 'stripe', 20.48, 'INR', 'success', NULL, NULL, '2025-12-10 16:27:39', '2025-12-10 16:27:39', 'stripe', 'visa', '1111', 'pi_3Scl5BCY24ISQw4B1wCTCsSB'),
(8, 28, 1, '', 'Card', 'stripe', 59.96, 'INR', 'success', NULL, NULL, '2025-12-12 11:10:10', '2025-12-12 11:10:10', 'stripe', 'visa', '1111', 'pi_3SdP4xCY24ISQw4B0aljV1Od'),
(9, 31, 1, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:14:50', '2025-12-12 17:14:50', 'stripe', 'visa', '1111', 'pi_3SdUlfCY24ISQw4B0o8K6Oqq'),
(10, 32, 1, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:15:40', '2025-12-12 17:15:40', 'stripe', 'visa', '1111', 'pi_3SdUlfCY24ISQw4B0o8K6Oqq'),
(11, 33, 1, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:27:49', '2025-12-12 17:27:49', 'stripe', 'visa', '1111', 'pi_3SdUlfCY24ISQw4B0o8K6Oqq'),
(12, 34, 6, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:42:31', '2025-12-12 17:42:31', 'stripe', 'visa', '1111', 'pi_3SdUuSCY24ISQw4B0LlsuAto'),
(13, 35, 6, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:45:59', '2025-12-12 17:45:59', 'stripe', '', '1111', 'pi_3SdUuSCY24ISQw4B0LlsuAto'),
(14, 36, 6, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-12 17:46:45', '2025-12-12 17:46:45', 'stripe', '', '1111', 'pi_3SdUuSCY24ISQw4B0LlsuAto'),
(16, 38, 1, '', 'Card', 'stripe', 13.99, 'INR', 'refunded', NULL, NULL, '2025-12-12 19:41:20', '2025-12-20 11:34:58', 'stripe', 'visa', '1111', 'pi_3SdX3gCY24ISQw4B1sGbbkn9'),
(17, 39, 5, '', 'Card', 'stripe', 33.98, 'INR', 'success', NULL, NULL, '2025-12-13 15:18:06', '2025-12-13 15:18:06', 'stripe', 'visa', '1111', 'pi_3SdpQXCY24ISQw4B0S6piie2'),
(18, 40, 5, '', 'Card', 'stripe', 38.95, 'INR', 'success', NULL, NULL, '2025-12-13 15:19:57', '2025-12-13 15:19:57', 'stripe', 'visa', '1111', 'pi_3SdpSDCY24ISQw4B0NCoy81a'),
(19, 41, 6, '', 'Card', 'stripe', 20.99, 'INR', 'success', NULL, NULL, '2025-12-18 18:17:08', '2025-12-18 18:17:08', 'stripe', 'visa', '1111', 'pi_3SdUuSCY24ISQw4B0LlsuAto'),
(20, 42, 6, '', 'Card', 'stripe', 33.98, 'INR', 'success', NULL, NULL, '2025-12-18 18:34:51', '2025-12-18 18:34:51', 'stripe', 'visa', '1111', 'pi_3SfgsUCY24ISQw4B1ttY6mlD'),
(21, 43, 6, '', 'Card', 'stripe', 108.00, 'INR', 'success', NULL, NULL, '2025-12-19 17:51:38', '2025-12-19 17:51:38', 'stripe', 'visa', '1111', 'pi_3Sg2gFCY24ISQw4B0NSjDeRV'),
(22, 44, 6, '', 'Card', 'stripe', 59.96, 'INR', 'refunded', NULL, NULL, '2025-12-19 18:23:51', '2025-12-19 18:24:48', 'stripe', 'visa', '1111', 'pi_3Sg3BVCY24ISQw4B0EtsWBWr'),
(23, 45, 1, '', 'Card', 'stripe', 33.98, 'INR', 'refunded', NULL, NULL, '2025-12-20 11:06:31', '2025-12-20 12:24:52', 'stripe', 'visa', '1111', 'pi_3SgIpdCY24ISQw4B0YGqHE0T'),
(24, 46, 1, '', 'Card', 'stripe', 33.98, 'INR', 'refunded', NULL, NULL, '2025-12-20 11:22:32', '2025-12-20 11:24:13', 'stripe', 'visa', '1111', 'pi_3SgJ5ACY24ISQw4B1Q9LZoAc'),
(25, 47, 1, '', 'Card', 'stripe', 60.92, 'INR', 'refunded', NULL, NULL, '2025-12-20 11:45:00', '2025-12-20 12:17:40', 'stripe', 'visa', '1111', 'pi_3SgJQxCY24ISQw4B19jJuhIs'),
(26, 48, 1, '', 'Card', 'stripe', 108.00, 'INR', 'success', NULL, NULL, '2025-12-20 11:56:43', '2025-12-20 11:56:43', 'stripe', 'visa', '1111', 'pi_3SgJcFCY24ISQw4B1LqOfbq0'),
(27, 49, 1, '', 'Card', 'stripe', 34.96, 'INR', 'success', NULL, NULL, '2025-12-20 11:58:45', '2025-12-20 11:58:45', 'stripe', 'visa', '1111', 'pi_3SgJeNCY24ISQw4B1qffHM57'),
(28, 50, 1, '', 'Card', 'stripe', 20.98, 'INR', 'success', NULL, NULL, '2025-12-20 12:05:06', '2025-12-20 12:05:06', 'stripe', 'visa', '1111', 'pi_3SgJkWCY24ISQw4B0dgYJMi3'),
(29, 51, 1, '', 'Card', 'stripe', 20.98, 'INR', 'success', NULL, NULL, '2025-12-20 12:10:07', '2025-12-20 12:10:07', 'stripe', 'visa', '1111', 'pi_3SgJpOCY24ISQw4B1eu1pydR'),
(30, 52, 1, '', 'Card', 'stripe', 63.99, 'INR', 'success', NULL, NULL, '2025-12-20 12:30:20', '2025-12-20 12:30:20', 'stripe', 'visa', '1111', 'pi_3SgK8wCY24ISQw4B1Qfy06nG'),
(31, 53, 7, '', 'Card', 'stripe', 37.95, 'INR', 'refunded', NULL, NULL, '2025-12-20 14:02:30', '2025-12-20 14:16:45', 'stripe', 'visa', '1111', 'pi_3SgLa2CY24ISQw4B0kXJvJwy'),
(32, 54, 7, '', 'Card', 'stripe', 19.98, 'INR', 'success', NULL, NULL, '2025-12-29 14:27:03', '2025-12-29 14:27:03', 'stripe', 'visa', '1111', 'pi_3SjcFeCY24ISQw4B1jpWMi1i'),
(33, 55, 7, '', 'Card', 'stripe', 21.98, 'INR', 'success', NULL, NULL, '2025-12-29 14:27:48', '2025-12-29 14:27:48', 'stripe', 'visa', '1111', 'pi_3SjcGZCY24ISQw4B1sQ02rYz'),
(34, 56, 1, '', 'Card', 'stripe', 24.98, 'INR', 'success', NULL, NULL, '2026-01-06 17:00:46', '2026-01-06 17:00:46', 'stripe', 'visa', '1111', 'pi_3SmYSzCY24ISQw4B1sSjTvfT'),
(35, 57, 1, '', 'Card', 'stripe', 19.32, 'INR', 'success', NULL, NULL, '2026-01-06 17:45:21', '2026-01-06 17:45:21', 'stripe', 'visa', '1111', 'pi_3SmZAACY24ISQw4B06k2Nwu5'),
(36, 58, 8, '', 'Card', 'stripe', 13.66, 'INR', 'refunded', NULL, NULL, '2026-01-07 10:52:09', '2026-01-07 10:54:08', 'stripe', 'visa', '1111', 'pi_3SmpB9CY24ISQw4B1UCmjp89'),
(37, 59, 8, '', 'Card', 'stripe', 58.00, 'INR', 'success', NULL, NULL, '2026-01-08 14:02:30', '2026-01-08 14:02:30', 'stripe', 'visa', '1111', 'pi_3SnEdZCY24ISQw4B1YoWdEyD'),
(38, 60, 8, '', 'Card', 'stripe', 13.66, 'INR', 'success', NULL, NULL, '2026-01-08 14:11:08', '2026-01-08 14:11:08', 'stripe', 'visa', '1111', 'pi_3SnElyCY24ISQw4B16QXGllr');

INSERT INTO `tbl_ratings_reviews` (`id`, `user_id`, `restaurant_id`, `order_id`, `item_id`, `rating`, `review`, `created_at`, `updated_at`) VALUES
(1, 1, 1, 27, '2,1', 4, 'Good food and fast delivery.', '2025-12-11 17:15:08', '2025-12-11 17:15:08'),
(2, 7, 1, 55, '2', 4, 'Good', '2026-01-06 10:26:33', '2026-01-06 10:26:33'),
(3, 7, 1, 54, '1', 3, 'Test', '2026-01-06 10:28:58', '2026-01-06 10:28:58'),
(4, 1, 1, 48, '6', 4, 'Testing oirder', '2026-01-06 10:31:47', '2026-01-06 10:31:47'),
(5, 1, 2, 32, '3', 4, 'Good', '2026-01-06 10:33:25', '2026-01-06 10:33:25');

INSERT INTO `users` (`id`, `name`, `email`, `phone`, `address`, `city`, `state`, `zipcode`, `country`, `gender`, `date_of_birth`, `user_type`, `profile_pic`, `id_number`, `id_doc_front`, `id_doc_back`, `is_active`, `created_at`, `updated_at`) VALUES
(7, 'John Doe', 'john@example.com', '+1 555-999-2222', 'pune', 'pune', 'maharastra', 4113003, 'United states', 'male', '1990-01-01', 'customer', 'http://localhost:8080/uploads/ids/1763036398830999000_images.jpg', 'A1234567', 'http://localhost:8080/uploads/ids/1763036398829289000_OnB1.jpg', 'http://localhost:8080/uploads/ids/1763036398830510000_OnB2.jpg', 1, '2025-11-13 17:49:58', '2025-11-13 17:49:58');



/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;