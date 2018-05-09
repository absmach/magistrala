import { Injectable } from '@angular/core';
import { Observable } from 'rxjs/Observable';

import { Client } from '../store/models';

const MOCK_CHANNELS = {
  channels: [
    {
      name: 'pera'
    },
    {
      name: 'dzoni'
    }
  ]
};

@Injectable()
export class MockChannelsService {
  getChannels() {
      return Observable.of(MOCK_CHANNELS).delay(1000);
  }

  addChannel(client: Client) {
    MOCK_CHANNELS.channels.push(client);
    return Observable.of(1).delay(1000);
  }
}
