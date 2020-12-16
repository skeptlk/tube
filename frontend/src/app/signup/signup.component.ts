import { Component } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { first } from 'rxjs/operators';
import { NotificationService, UserService } from '../services';

@Component({
    selector: 'signup',
    templateUrl: './signup.component.html',
    styleUrls: ['./signup.component.scss']
})
export class SignupComponent {

    signupForm = new FormGroup({
        name: new FormControl('', [Validators.required]),
        email: new FormControl('', [Validators.required, Validators.email]),
        password: new FormControl('', [Validators.required, Validators.minLength(4)])
    })

    constructor(
        private notifService: NotificationService,
        private userService: UserService,
        private router: Router,
    ) { }

    submit() {
        if (this.signupForm.valid) {
            this.userService.create(this.signupForm.value)
                .pipe(first())
                .subscribe(
                    () => {
                        this.notifService.message("Your account has been created successfully!");
                        return this.router.navigate(['login']);
                    },
                    (error) => this.notifService.error(error)
                );
        }
        
    }
}
