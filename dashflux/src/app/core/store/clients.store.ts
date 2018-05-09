import { Injectable } from '@angular/core';
import { action, observable } from 'mobx';

import { ClientsService } from '../services/clients/clients.service';
import { Client } from './models';
import { UiStore } from './ui.store';

@Injectable()
export class ClientsStore {
    @observable clients: Client[] = [];

    constructor(
        private uiState: UiStore,
        private clientsService: ClientsService,
    ) { }

    @action
    getClients() {
        this.uiState.loading = true;
        this.clientsService.getClients()
            .subscribe((payload: any) => {
                this.uiState.loading = false;
                this.clients = payload.clients;
            }, () => {
                this.uiState.loading = false;
            });
    }

    @action
    addClient(client: Client) {
        this.uiState.loading = true;
        this.clientsService.addClient(client)
            .subscribe(() => {
                this.uiState.loading = false;
                this.getClients();
            }, () => {
                this.uiState.loading = false;
            });
    }

    @action
    editClient(client: Client) {
        this.uiState.loading = true;
        this.clientsService.editClient(client)
            .subscribe(() => {
                this.uiState.loading = false;
                this.getClients();
            }, () => {
                this.uiState.loading = false;
            });
    }

    @action
    deleteClient(client: Client) {
        this.uiState.loading = true;
        this.clientsService.deleteClient(client)
            .subscribe(() => {
                this.uiState.loading = false;
                this.getClients();
            }, () => {
                this.uiState.loading = false;
            });
    }
}
