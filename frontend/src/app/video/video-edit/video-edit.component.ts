import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { VideoService } from 'src/app/services';
import { Category, Video } from '../../models';


@Component({
    selector: 'video-edit',
    templateUrl: './video-edit.component.html',
    styleUrls: ['./video-edit.component.scss']
})
export class VideoEditComponent implements OnInit {
    categories: Category[] = [];
    selectedCategories = [];
    form: FormGroup = new FormGroup({
        title: new FormControl('', [Validators.required]),
        description: new FormControl('')
    });
    video: Video;

    constructor(
        private videoService: VideoService,
        private route: ActivatedRoute,
        private router: Router
    ) { }

    ngOnInit(): void {
        this.route.paramMap.subscribe(async params => {
            const id = +params.get('id');

            this.categories = await this.videoService.getAllCategories().toPromise();
            this.video = await this.videoService.getInfo(id).toPromise();
            this.form.setValue({
                title: this.video.title,
                description: this.video.description
            });
            this.selectedCategories = this.video.categories.map(c => c.id);
        })
    }

    submit(e) {
        e.preventDefault();
        this.video.title = this.form.get('title').value;
        this.video.description = this.form.get('description').value;
        this.video.categoryIds = this.selectedCategories.map(cat => +cat);

        console.log(this.selectedCategories);

        this.videoService.update(this.video).subscribe(res => {
            this.router.navigate(['v', this.video.id]);
        })
    }

}
