## Copyright (c) Abstract Machines
## SPDX-License-Identifier: Apache-2.0

## Script used to generate random data to do testing the timescaledb "messages" table
## The script generate CSV file, which can be loaded into data base using the below command
### psql command to copy csv
### psql -h 127.0.0.1 -U supermq -d supermq -p 5433 -W -c  "\COPY messages (time, channel, subtopic, publisher, protocol, name, unit, value) FROM 'scripts/data-gen/messages.csv' WITH (FORMAT csv, HEADER)"

import random
import uuid
from datetime import datetime, timedelta
import os

num_channels=100
num_subtopic=1 ## subtopic per channel
num_publisher=3 ## publisher per subtopic
num_metrics=10 ## metrics per publisher
last_num_days=30
data_interval_minutes=15

# Prepare rows (all unique channel, subtopic, publisher, name combinations)


print(f"{num_channels} channels")
print(f"{num_subtopic} subtopics (per channel)")
print(f"{num_publisher} publishers (per subtopic)")
print(f"{num_metrics} names (per publisher)")
rows = []
for channel_idx in range(1, num_channels+1):  # (num_channels) channels
    channel_id = uuid.uuid4()
    for subtopic_idx in range(1, num_subtopic+1):  # (num_subtopic_per_channel) subtopics per channel
        subtopic = f"subtopic_{subtopic_idx}"
        for publisher_idx in range(1, num_publisher+1):  # (num_publisher_per_subtopic) publishers per subtopic
            publisher_id = uuid.uuid4()
            for name_idx in range(1, num_metrics+1):  # (num_metrics_per_publisher) names per publisher
                name = f"metric_{name_idx}"
                rows.append({
                    "channel": channel_id,
                    "subtopic": subtopic,
                    "publisher": publisher_id,
                    "name": name,
                })

print(f"Total unique metric: {len(rows)}")

start_time = datetime.now() - timedelta(days=last_num_days)
interval = timedelta(minutes=data_interval_minutes)
num_intervals = int((last_num_days * 24 * 60) / data_interval_minutes)  # number of data_interval_minutes in last_num_days

print(f"generating data for last {last_num_days} days with data interval of {data_interval_minutes} minutes which is {num_intervals} timestamps")

script_directory = os.path.dirname(os.path.abspath(__file__))

data_file_path=f"{script_directory}/messages.csv"


# Open CSV file for writing
with open(data_file_path, 'w') as f:
    f.write("time,channel,subtopic,publisher,protocol,name,unit,value\n")

    for i in range(num_intervals):
        timestamp = start_time + (i * interval)
        timestamp_ns = int(timestamp.timestamp() * 1_000_000_000)  # nanoseconds
        for row in rows:
            value = random.uniform(0, 100)  # generate a float64 value between 0 and 100
            f.write(f"{timestamp_ns},{row['channel']},{row['subtopic']},{row['publisher']},mqtt,{row['name']},unit,{value}\n")

print(f"Finished writing CSV at {data_file_path} with {len(rows) * num_intervals} rows = {len(rows)} unique metrics (channel + subtopic + publisher + metric) x {num_intervals} timestamps")


