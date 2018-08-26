/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import { HttpClientModule } from '@angular/common/http';
import { async, TestBed } from '@angular/core/testing';
import { RouterTestingModule } from '@angular/router/testing';

import { AppComponent } from './app.component';
import { MaterialModule } from './core/material/material.module';
import { AuthenticationService } from './core/services/auth/authentication.service';
import { TokenStorage } from './core/services/auth/token-storage.service';
import { ChannelsService } from './core/services/channels/channels.service';
import { ThingsService } from './core/services/things/things.service';
import { UiStore } from './core/store/ui.store';
import { AuthStore } from './core/store/auth.store';

describe('AppComponent', () => {
  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [
        AppComponent
      ],
      imports: [
        MaterialModule,
        HttpClientModule,
        RouterTestingModule
      ],
      providers: [
        UiStore,
        AuthStore,
        AuthenticationService,
        TokenStorage,
        ThingsService,
        ChannelsService,
      ]
    }).compileComponents();
  }));
  it('should create the app', async(() => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.debugElement.componentInstance;
    expect(app).toBeTruthy();
  }));
});
