import { Component, EventEmitter, Inject, OnInit, Output } from '@angular/core';
import { AbstractControl, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';

import { Thing } from '../../../core/store/models';

@Component({
  selector: 'app-thing-dialog',
  templateUrl: './thing-dialog.component.html',
  styleUrls: ['./thing-dialog.component.scss']
})
export class ThingDialogComponent implements OnInit {
  addThingForm: FormGroup;
  @Output() submit: EventEmitter<Thing> = new EventEmitter<Thing>();
  editMode: boolean;

  constructor(
    private fb: FormBuilder,
    private dialogRef: MatDialogRef<ThingDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: Thing
  ) { }

  ngOnInit() {
    this.addThingForm = this.fb.group(
      {
        id: null,
        type: ['', [Validators.required]],
        name: ['', [Validators.required, Validators.minLength(5)]],
        metadata: ['']
      }
    );

    if (this.data) {
      this.editMode = true;
      this.addThingForm.patchValue(this.data);
    }
  }

  onAddThing() {
    const thing = this.addThingForm.value;
    this.submit.emit(thing);
    this.dialogRef.close();
  }
}
