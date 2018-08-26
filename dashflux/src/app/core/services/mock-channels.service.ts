/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs/Observable';

import { Thing } from '../store/models';

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

  addChannel(client: Thing) {
    MOCK_CHANNELS.channels.push(client);
    return Observable.of(1).delay(1000);
  }
}
