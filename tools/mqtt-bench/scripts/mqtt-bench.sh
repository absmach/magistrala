#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

i=0
echo "BEGIN TEST " > result.$1.out
for mtls in true 
do
	for ret in false true
	do
		for qos in 0 1 2
		do
		for pub in 1 10 100
		do
			for sub in 1 10 
			do
				for message in 100 1000
				do
					if [[ $pub -eq 100 && $message -eq 1000 ]];
					then
						continue
					fi
						
					for size in 100 500
					do
						let "i += 1"
						echo "=================================TEST $i=========================================" >> $1-$i.out
						echo "MTLS: $mtls RETAIN: $ret, QOS $qos" >> $1-$i.out
						echo "Pub:" $pub ", Sub:" $sub ", MsgSize:" $size ", MsgPerPub:" $message    >> $1-$i.out
						echo "=================================================================================" >> $1-$i.out
						if [ "$mtls" = true ];
						then
							echo "| " >> $1-$i.out
							echo "| ./mqtt-bench --channels $3 -s $size -n $message  --subs $sub --pubs $pub  -q $qos --retain=$ret -m=true -b tcps://$2:8883 --quiet=true --ca ../../../docker/ssl/certs/ca.crt -t=true" >> $1-$i.out
							echo "| " >> $1-$i.out
							../cmd/mqtt-bench --channels $3 -s $size -n $message  --subs $sub --pubs $pub  -q $qos --retain=$ret -m=true -b tcps://$2:8883 --quiet=true --ca ../../../docker/ssl/certs/ca.crt -t=true >> $1-$i.out
						else
							echo "| " >> $1-$i.out
							echo "| ./mqtt-bench --channels $3 -s $size -n $message  --subs $sub --pubs $pub  -q $qos  --retain=$ret -b tcp://$2:1883 --quiet=true" >> $1-$i.out	
							echo "| " >> $1-$i.out
							../cmd/mqtt-bench --channels $3 -s $size -n $message  --subs $sub --pubs $pub  -q $qos  --retain=$ret -b tcp://$2:1883 --quiet=true >> $1-$i.out
						fi
						sleep 2
					done
				done
			done
		done
		done

	done
done 
files=`ls test*.out | sort --version-sort `
for file in $files
do
	cat $file >> result.$1.out
done
echo "END TEST " >> result.$1.out
