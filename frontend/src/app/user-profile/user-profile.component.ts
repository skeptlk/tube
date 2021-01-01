import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { User, Video } from '../models';
import { UserService, VideoService } from '../services';

@Component({
  selector: 'user-profile',
  templateUrl: './user-profile.component.html',
  styleUrls: ['./user-profile.component.scss']
})
export class UserProfileComponent implements OnInit {
    user: User;
    videos: Video[] = [];

    editProfileForm = new FormGroup({
        name: new FormControl('', [Validators.required]),
        email: new FormControl('', [Validators.required, Validators.email])
    })

    constructor(
        private userService: UserService,
        public vidService: VideoService,
        private route: ActivatedRoute
    ) { }

    async ngOnInit() {
        this.route.paramMap.subscribe(async params => {
            var id = +params.get('id');
            console.log(id);
            
            this.user = await this.userService.get(id).toPromise();
            this.videos = await this.vidService.getUserVideos(this.user.id).toPromise();
        })
    }

}
