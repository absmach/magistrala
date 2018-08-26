/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs/Observable';

import { Thing } from '../store/models';

const MOCK_THINGS = [
];

@Injectable()
export class MockThingsService {
  getThings() {
      return Observable.of(MOCK_THINGS).delay(1000);
  }

  addThing(thing: Thing) {
    MOCK_THINGS.push(thing);
    return Observable.of(1).delay(1000);
  }
}
