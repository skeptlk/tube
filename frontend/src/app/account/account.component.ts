import { Component, OnInit } from '@angular/core';
import { AuthService, UserService, VideoService } from '../services';
import { User, Video } from '../models';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';

@Component({
    selector: 'account',
    templateUrl: './account.component.html',
    styleUrls: ['./account.component.scss']
})
export class AccountComponent implements OnInit {
    user: User;
    myVideos: Video[] = [];

    editProfileForm = new FormGroup({
        name: new FormControl('', [Validators.required]),
        email: new FormControl('', [Validators.required, Validators.email])
    })

    constructor(
        public auth: AuthService,
        // private userService: UserService,
        public vidService: VideoService,
        private router: Router
    ) { }

    ngOnInit(): void {
        this.user = this.auth.currentUserValue;
        this.editProfileForm.patchValue({
            name: this.user.name,
            email: this.user.email
        });
        this.vidService
            .getUserVideos(this.user.id)
            .subscribe(vids => {
                this.myVideos = vids;
            });
    }

    logout() {
        this.auth.logout();
        this.router.navigate(['']);
    }

    editProfile() {
        // ...
    }

}
