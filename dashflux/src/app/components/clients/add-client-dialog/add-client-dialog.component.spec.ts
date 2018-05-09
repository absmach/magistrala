import { HttpClientModule } from '@angular/common/http';
import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RouterTestingModule } from '@angular/router/testing';

import { MaterialModule } from '../../../core/material/material.module';
import { AuthenticationService } from '../../../core/services/auth/authentication.service';
import { TokenStorage } from '../../../core/services/auth/token-storage.service';
import { ChannelsService } from '../../../core/services/channels/channels.service';
import { ClientsService } from '../../../core/services/clients/clients.service';
import { AddClientDialogComponent } from './add-client-dialog.component';

describe('AddClientDialogComponent', () => {
  let component: AddClientDialogComponent;
  let fixture: ComponentFixture<AddClientDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ AddClientDialogComponent ],
      imports: [
        MaterialModule,
        MatDialogModule,
        HttpClientModule,
        RouterTestingModule,
        FormsModule,
        ReactiveFormsModule,
        NoopAnimationsModule
      ],
      providers: [
        AuthenticationService,
        TokenStorage,
        ClientsService,
        ChannelsService,
        { provide: MatDialogRef, useValue: {} },
        { provide: MAT_DIALOG_DATA, useValue: [] },
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(AddClientDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
