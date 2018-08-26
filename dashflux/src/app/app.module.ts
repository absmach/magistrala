/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import './rxjs-extensions.ts';
import 'hammerjs';

import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { APP_BASE_HREF } from '@angular/common';
import { NgModule } from '@angular/core';
import { FlexLayoutModule } from '@angular/flex-layout';
import { ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { MobxAngularModule } from 'mobx-angular';
import { AUTH_SERVICE, AuthModule, PROTECTED_FALLBACK_PAGE_URI, PUBLIC_FALLBACK_PAGE_URI } from 'ngx-auth';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { LoginComponent } from './components/auth/login/login.component';
import { SignupComponent } from './components/auth/signup/signup.component';
import { ChannelDialogComponent } from './components/channels/channel-dialog/channel-dialog.component';
import { ChannelsComponent } from './components/channels/channels.component';
import { ThingDialogComponent } from './components/things/thing-dialog/thing-dialog.component';
import { ThingsComponent } from './components/things/things.component';
import { ConfirmationDialogComponent } from './components/shared/confirmation-dialog/confirmation-dialog.component';
import { MaterialModule } from './core/material/material.module';
import { AuthenticationService } from './core/services/auth/authentication.service';
import { TokenStorage } from './core/services/auth/token-storage.service';
import { ChannelsService } from './core/services/channels/channels.service';
import { ThingsService } from './core/services/things/things.service';
import { MockAuthService } from './core/services/mock-auth.service';
import { MockChannelsService } from './core/services/mock-channels.service';
import { MockThingsService } from './core/services/mock-things.service';
import { ChannelsStore } from './core/store/channels.store';
import { ThingsStore } from './core/store/things.store';
import { UiStore } from './core/store/ui.store';
import { AuthStore } from './core/store/auth.store';
import { UnauthorizedInterceptor } from './core/services/auth/unauthorized.interceptor';

export function factory(authenticationService: AuthenticationService) {
  return authenticationService;
}

@NgModule({
  declarations: [
    AppComponent,
    ThingsComponent,
    ChannelsComponent,
    SignupComponent,
    LoginComponent,
    ThingDialogComponent,
    ConfirmationDialogComponent,
    ChannelDialogComponent,
  ],
  imports: [
    AuthModule,
    BrowserModule,
    BrowserAnimationsModule,
    AppRoutingModule,
    HttpClientModule,
    MaterialModule,
    FlexLayoutModule,
    ReactiveFormsModule,
    MobxAngularModule,
  ],
  providers: [
    UiStore,
    ThingsStore,
    ChannelsStore,
    AuthStore,
    MockAuthService,
    MockThingsService,
    MockChannelsService,
    ThingsService,
    ChannelsService,
    TokenStorage,
    AuthenticationService,
    { provide: PROTECTED_FALLBACK_PAGE_URI, useValue: '/' },
    { provide: PUBLIC_FALLBACK_PAGE_URI, useValue: '/login' },
    {
      provide: AUTH_SERVICE,
      deps: [AuthenticationService],
      useFactory: factory
    },
    { provide: HTTP_INTERCEPTORS, useClass: UnauthorizedInterceptor, multi: true },
    { provide: APP_BASE_HREF, useValue: '/app/' }
  ],
  bootstrap: [AppComponent],
  entryComponents: [
    ThingDialogComponent,
    ChannelDialogComponent,
    ConfirmationDialogComponent
  ]
})
export class AppModule { }
