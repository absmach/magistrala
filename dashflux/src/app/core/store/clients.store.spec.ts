import { HttpClientModule } from '@angular/common/http';
import { inject, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';
import { toJS } from 'mobx';
import { Observable } from 'rxjs/Observable';

import { ClientsService } from '../services/clients/clients.service';
import { ClientsStore } from './clients.store';
import { Client } from './models';
import { UiStore } from './ui.store';

describe('ClientsStore', () => {
    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [
                HttpClientModule,
                RouterTestingModule.withRoutes([])
            ],
            providers: [
                ClientsStore,
                UiStore,
                ClientsService,
            ]
        });
    });

    it('should be created', inject([ClientsStore], (clientsStore: ClientsStore) => {
        expect(clientsStore).toBeTruthy();
    }));

    describe('getClients', () => {
        it('should set the loading flag to true before service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const getClients = spyOn(clientsService, 'getClients').and.returnValue({ subscribe: () => { } });

                clientsStore.getClients();

                expect(uiStore.loading).toBeTruthy();
            }));

        it('should set the loading flag to false after successful get', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const getClients = spyOn(clientsService, 'getClients').and.returnValue(Observable.of(true));

                clientsStore.getClients();

                expect(uiStore.loading).toBeFalsy();
            }));

        it('should set the clients property to the returned clients from the service', inject([ClientsStore, ClientsService],
            (clientsStore: ClientsStore, clientsService: ClientsService) => {
                const serviceReturnValue = { clients: [] };
                const getChannels = spyOn(clientsService, 'getClients').and.returnValue(Observable.of(serviceReturnValue));

                clientsStore.getClients();

                expect(toJS(clientsStore.clients)).toEqual(serviceReturnValue.clients);
            }));

        it('should set the loading flag to false after failed get', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const getClients = spyOn(clientsService, 'getClients').and.returnValue(Observable.throw(''));

                clientsStore.getClients();

                expect(uiStore.loading).toBeFalsy();
            }));
    });

    describe('addClient', () => {
        it('should set the loading flag to true before service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const addClient = spyOn(clientsService, 'addClient').and.returnValue({ subscribe: () => { } });
                const newClient: Client = {
                    name: 'new client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.addClient(newClient);

                expect(uiStore.loading).toBeTruthy();
            }));

        it('should set the loading flag to false after successful service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const addClient = spyOn(clientsService, 'addClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();
                const newClient: Client = {
                    name: 'new client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.addClient(newClient);

                expect(uiStore.loading).toBeFalsy();
            }));

        it('should call the ClientsStore.getClients after successful add', inject([ClientsStore, ClientsService],
            (clientsStore: ClientsStore, clientsService: ClientsService) => {
                const addClient = spyOn(clientsService, 'addClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();

                const newClient: Client = {
                    name: 'new client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.addClient(newClient);

                expect(storeGetClientsSpy).toHaveBeenCalled();
            }));

        it('should set the loading flag to false after failed add', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const addClient = spyOn(clientsService, 'addClient').and.returnValue(Observable.throw(''));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();

                const newClient: Client = {
                    name: 'new client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.addClient(newClient);

                expect(uiStore.loading).toBeFalsy();
            }));
    });

    describe('editClient', () => {
        it('should set the loading flag to true before service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const editClient = spyOn(clientsService, 'editClient').and.returnValue({ subscribe: () => { } });
                const editedClient: Client = {
                    name: 'edited client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.editClient(editedClient);

                expect(uiStore.loading).toBeTruthy();
            }));

        it('should set the loading flag to false after successful service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const editClient = spyOn(clientsService, 'editClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();
                const editedClient: Client = {
                    name: 'edited client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.editClient(editedClient);

                expect(uiStore.loading).toBeFalsy();
            }));

        it('should call the ClientsStore.getChannels after successful edit', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const editClient = spyOn(clientsService, 'editClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();

                const editedClient: Client = {
                    name: 'edited client',
                    type: 'app',
                    payload: '',
                };


                clientsStore.editClient(editedClient);

                expect(storeGetClientsSpy).toHaveBeenCalled();
            }));

        it('should set the loading flag to false after failed edit', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const editClient = spyOn(clientsService, 'editClient').and.returnValue(Observable.throw(''));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();

                const editedClient: Client = {
                    name: 'edited client',
                    type: 'app',
                    payload: '',
                };

                clientsStore.editClient(editedClient);

                expect(uiStore.loading).toBeFalsy();
            }));
    });

    describe('deleteClient', () => {
        it('should set the loading flag to true before service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const deleteClient = spyOn(clientsService, 'deleteClient').and.returnValue({ subscribe: () => { } });
                const clientToBeDeleted: Client = {
                    name: 'clientToBeDeleted',
                    type: 'app',
                    payload: ''
                };

                clientsStore.deleteClient(clientToBeDeleted);

                expect(uiStore.loading).toBeTruthy();
            }));

        it('should set the loading flag to false after successful service call', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const deleteClient = spyOn(clientsService, 'deleteClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();
                const clientToBeDeleted: Client = {
                    name: 'clientToBeDeleted',
                    type: 'app',
                    payload: ''
                };


                clientsStore.deleteClient(clientToBeDeleted);

                expect(uiStore.loading).toBeFalsy();
            }));

        it('should call the clientsStore.getChannels after successful add', inject([ClientsStore, ClientsService],
            (clientsStore: ClientsStore, clientsService: ClientsService) => {
                const deleteClient = spyOn(clientsService, 'deleteClient').and.returnValue(Observable.of(true));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();

                const clientToBeDeleted: Client = {
                    name: 'clientToBeDeleted',
                    type: 'app',
                    payload: ''
                };

                clientsStore.deleteClient(clientToBeDeleted);

                expect(storeGetClientsSpy).toHaveBeenCalled();
            }));

        it('should set the loading flag to false after failed delete', inject([ClientsStore, UiStore, ClientsService],
            (clientsStore: ClientsStore, uiStore: UiStore, clientsService: ClientsService) => {
                const deleteClient = spyOn(clientsService, 'deleteClient').and.returnValue(Observable.throw(''));
                const storeGetClientsSpy = spyOn(clientsStore, 'getClients').and.stub();
                const clientToBeDeleted: Client = {
                    name: 'clientToBeDeleted',
                    type: 'app',
                    payload: ''
                };


                clientsStore.deleteClient(clientToBeDeleted);

                expect(uiStore.loading).toBeFalsy();
            }));
    });
});
