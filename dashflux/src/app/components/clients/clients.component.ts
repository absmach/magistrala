import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material';
import { toJS } from 'mobx';
import { Observable } from 'rxjs/Observable';

import { Client } from '../../core/store/models';
import { ConfirmationDialogComponent } from '../shared/confirmation-dialog/confirmation-dialog.component';
import { AddClientDialogComponent } from './add-client-dialog/add-client-dialog.component';
import { ClientsStore } from '../../core/store/clients.store';
import { ChannelsStore } from '../../core/store/channels.store';

@Component({
  selector: 'app-clients',
  templateUrl: './clients.component.html',
  styleUrls: ['./clients.component.scss']
})
export class ClientsComponent implements OnInit {
  clients: Observable<Client[]>;
  displayedColumns = ['id', 'name', 'type', 'payload', 'actions'];

  constructor(
    private dialog: MatDialog,
    public clientsStore: ClientsStore,
    public channelsStore: ChannelsStore,
  ) { }

  ngOnInit() {
    this.clientsStore.getClients();
    this.channelsStore.getChannels();
  }

  addClient() {
    const dialogRef = this.dialog.open(AddClientDialogComponent);

    dialogRef.componentInstance.submit.subscribe((client: Client) => {
      this.clientsStore.addClient(client);
    });
  }

  editClient(client: Client) {
    const dialogRef = this.dialog.open(AddClientDialogComponent, {
      data: client
    });

    dialogRef.componentInstance.submit.subscribe((editedClient: Client) => {
      this.clientsStore.editClient(toJS(editedClient));
    });
  }

  deleteClient(client: Client) {
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        question: 'Are you sure you want to delete the client?'
      }
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.clientsStore.deleteClient(toJS(client));
      }
    });
  }
}
