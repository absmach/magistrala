
# Linux install


Copy senmlserver.service to /etc/systemd/system
copy senmlServer to /usr/bin

Edit up the service file
sudo systemctl daemon-reload
sudo systemctl enable senmlserver.service
sudo systemctl start senmlserver.service

Enable firewall
sudo ufw allow 8880/tcp

Test with
curl -d '[ { "n":"junk1", "v":12.4 , "u":"V" } ] ' http://10.1.3.17:8880


influx -host 10.1.3.17
use fh2
select * from senml

