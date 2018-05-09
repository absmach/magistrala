import './rxjs-extensions.ts';
import 'hammerjs';

import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
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
import { AddChannelDialogComponent } from './components/channels/add-channel-dialog/add-channel-dialog.component';
import { ChannelsComponent } from './components/channels/channels.component';
import { AddClientDialogComponent } from './components/clients/add-client-dialog/add-client-dialog.component';
import { ClientsComponent } from './components/clients/clients.component';
import { ConfirmationDialogComponent } from './components/shared/confirmation-dialog/confirmation-dialog.component';
import { MaterialModule } from './core/material/material.module';
import { AuthenticationService } from './core/services/auth/authentication.service';
import { TokenStorage } from './core/services/auth/token-storage.service';
import { ChannelsService } from './core/services/channels/channels.service';
import { ClientsService } from './core/services/clients/clients.service';
import { MockAuthService } from './core/services/mock-auth.service';
import { MockChannelsService } from './core/services/mock-channels.service';
import { MockClientsService } from './core/services/mock-clients.service';
import { ChannelsStore } from './core/store/channels.store';
import { ClientsStore } from './core/store/clients.store';
import { UiStore } from './core/store/ui.store';
import { AuthStore } from './core/store/auth.store';
import { UnauthorizedInterceptor } from './core/services/auth/unauthorized.interceptor';

export function factory(authenticationService: AuthenticationService) {
  return authenticationService;
}

@NgModule({
  declarations: [
    AppComponent,
    ClientsComponent,
    ChannelsComponent,
    SignupComponent,
    LoginComponent,
    AddClientDialogComponent,
    ConfirmationDialogComponent,
    AddChannelDialogComponent,
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
    ClientsStore,
    ChannelsStore,
    AuthStore,
    MockAuthService,
    MockClientsService,
    MockChannelsService,
    ClientsService,
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
  ],
  bootstrap: [AppComponent],
  entryComponents: [
    AddClientDialogComponent,
    AddChannelDialogComponent,
    ConfirmationDialogComponent
  ]
})
export class AppModule { }
