CREATE TABLE Users (
    UserID     STRING(36) NOT NULL,
    Email      STRING(320) NOT NULL,
    FirstName  STRING(100) NOT NULL,
    LastName   STRING(100) NOT NULL,
    Status     STRING(20) NOT NULL,
    CreatedAt  TIMESTAMP NOT NULL,
    UpdatedAt  TIMESTAMP NOT NULL,
) PRIMARY KEY (UserID);

CREATE UNIQUE INDEX UsersByEmail ON Users(Email);

CREATE TABLE Orders (
    OrderID       STRING(36) NOT NULL,
    UserID        STRING(36) NOT NULL,
    Status        STRING(20) NOT NULL,
    TotalAmount   INT64 NOT NULL,
    TotalCurrency STRING(3) NOT NULL,
    CreatedAt     TIMESTAMP NOT NULL,
    UpdatedAt     TIMESTAMP NOT NULL,
) PRIMARY KEY (OrderID);

CREATE INDEX OrdersByUserID ON Orders(UserID);

CREATE TABLE OrderItems (
    OrderID     STRING(36) NOT NULL,
    ItemIndex   INT64 NOT NULL,
    ProductID   STRING(36) NOT NULL,
    ProductName STRING(200) NOT NULL,
    Quantity    INT64 NOT NULL,
    UnitAmount  INT64 NOT NULL,
    Currency    STRING(3) NOT NULL,
) PRIMARY KEY (OrderID, ItemIndex),
  INTERLEAVE IN PARENT Orders ON DELETE CASCADE;
