#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

####
## The script helps to copy tables from one database to another database
## This script can be used to synchronize domains and domains roles tables to groups, channels and clients database
## and groups and groups roles to channels and clients database
####

set -e
set -o pipefail


# Define source and target database connection details
SRC_DB_HOST="localhost"
SRC_DB_PORT="6003"
SRC_DB_USER="supermq"
SRC_DB_PASSWORD="supermq"
SRC_DB_NAME="domains"
TABLENAME_PREFIX=domains

DEST_DB_HOST="localhost"
DEST_DB_PORT="6005"
DEST_DB_USER="supermq"
DEST_DB_PASSWORD="supermq"
DEST_DB_NAME="channels"

# List of tables to copy
TABLES=("$TABLENAME_PREFIX" $TABLENAME_PREFIX"_roles" $TABLENAME_PREFIX"_role_actions" $TABLENAME_PREFIX"_role_members" )


# Loop through each table and copy data
for TABLE_NAME in "${TABLES[@]}"; do
    echo "Copying data from $SRC_DB_NAME.$TABLE_NAME to $DEST_DB_NAME.$TABLE_NAME..."

    # Set the source password and execute the COPY command
    PGPASSWORD="$SRC_DB_PASSWORD" psql -h "$SRC_DB_HOST" -p "$SRC_DB_PORT" -U "$SRC_DB_USER" -d "$SRC_DB_NAME" -c "COPY $TABLE_NAME TO STDOUT" | \
    # Set the target password and execute the TRUNCATE table and COPY commands
    PGPASSWORD="$DEST_DB_PASSWORD" psql -h "$DEST_DB_HOST" -p "$DEST_DB_PORT" -U "$DEST_DB_USER" -d "$DEST_DB_NAME" -c "TRUNCATE TABLE $TABLE_NAME CASCADE; COPY $TABLE_NAME FROM STDIN"

    # Check for errors
    if [ $? -ne 0 ]; then
        echo "Error: Failed copy data for table $TABLE_NAME. Exiting."
        exit 1
    fi

    echo "Table $TABLE_NAME successfully copied."
done

echo "All tables copied successfully!"
