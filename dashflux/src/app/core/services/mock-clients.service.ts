import { Injectable } from '@angular/core';
import { Observable } from 'rxjs/Observable';

import { Client } from '../store/models';

let MOCK_CLIENTS = [
];

@Injectable()
export class MockClientsService {
  getClients() {
      return Observable.of(MOCK_CLIENTS).delay(1000);
  }

  addClient(client: Client) {
    MOCK_CLIENTS.push(client);
    return Observable.of(1).delay(1000);
  }
}
