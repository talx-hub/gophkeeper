BEGIN TRANSACTION;
    INSERT INTO data_types(id, name)
    VALUES  (1, 'DataTypeAuthenticationCredentials'),
            (2, 'DataTypeCard'),
            (3, 'DataTypeBinary');
COMMIT;
