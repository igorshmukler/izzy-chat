CREATE DATABASE chat
GO

USE chat
GO

IF OBJECT_ID('dbo.Users', 'U') IS NOT NULL
   DROP TABLE dbo.Users
GO

CREATE TABLE dbo.Users
(
   Id            INT IDENTITY(1,1) PRIMARY KEY,
   Username      [NVARCHAR](50)  NOT NULL UNIQUE, -- need index
   PasswordHash  [VARCHAR](255)  NOT NULL, -- don't need NVARCHAR, since the name is base64 encoded, VARCHAR with one byte is enough
   Email         [NVARCHAR](50) UNIQUE
);
GO

IF OBJECT_ID('dbo.Messages', 'U') IS NOT NULL
   DROP TABLE dbo.Messages
GO

CREATE TABLE dbo.Messages
(
   Id            INT IDENTITY(1,1) PRIMARY KEY,
   Username      [NVARCHAR](50)  NOT NULL, -- need index
   MessageType   [VARCHAR](50)  NOT NULL,
   Recepient     [NVARCHAR](50)  NOT NULL,
   Payload       [NVARCHAR](255),
   Stamp         DATETIME2 -- need index
);

GO
IF OBJECT_ID('dbo.Channels', 'U') IS NOT NULL
   DROP TABLE dbo.Channels
GO

CREATE TABLE dbo.Channels
(
   Id            INT IDENTITY(1,1) PRIMARY KEY,
   FriendlyName  [NVARCHAR](50) NOT NULL UNIQUE,
   ChannelType   INT NOT NULL,
   OwnerId       INT NOT NULL,
   LongDescription [NVARCHAR](255),
   FOREIGN KEY(OwnerId) REFERENCES dbo.Users(Id)
);
GO

IF OBJECT_ID('dbo.ChannelMembers', 'U') IS NOT NULL
   DROP TABLE dbo.ChannelMembers
GO

CREATE TABLE dbo.ChannelMembers
(
   Id            INT IDENTITY(1,1) PRIMARY KEY,
   ChannelId     INT NOT NULL,
   UserId        INT NOT NULL,
   FOREIGN KEY(ChannelId) REFERENCES dbo.Channels(Id),
   FOREIGN KEY(UserId) REFERENCES dbo.Users(Id)
);
GO

IF OBJECT_ID('dbo.Sessions', 'U') IS NOT NULL
   DROP TABLE dbo.Sessions
GO

CREATE TABLE dbo.Sessions
(
   Id            INT IDENTITY(1,1) PRIMARY KEY,
   Username      [NVARCHAR](50)  NOT NULL UNIQUE,
   Token         [VARCHAR](50)  NOT NULL,
   -- FOREIGN KEY(Username) REFERENCES dbo.Users(Username)
);
GO
