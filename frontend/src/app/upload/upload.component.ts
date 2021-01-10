import { Component, ElementRef, HostListener, OnInit, ViewChild } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { Category } from '../models/category';
import { AuthService, VideoService } from '../services';

@Component({
    selector: 'app-upload',
    templateUrl: './upload.component.html',
    styleUrls: ['./upload.component.scss']
})
export class UploadComponent implements OnInit {

    @ViewChild("fileInp") fileInput: ElementRef;
    @HostListener('window:dragover', ['$event'])
    dragover(e) { e.preventDefault(); }
    @HostListener('window:drop', ['$event'])
    drop(e) { e.preventDefault(); }

    video: File = null;
    categories: Category[] = [];
    selectedCategories: Category[] = [];
    uploadForm = new FormGroup({
        title: new FormControl('', [Validators.required]),
        description: new FormControl(''),
        categories: new FormControl('')
    })
    dragActive: boolean = false;

    constructor(
        private vidService: VideoService,
        private auth: AuthService,
        private router: Router
    ) { }

    async ngOnInit() {
        this.categories = await this.vidService.getAllCategories().toPromise();
    }

    onFileSelected(files: FileList) {
        if (files.length > 0)
            this.video = files.item(0);
    }

    selectFile() {
        this.fileInput.nativeElement.click();
    }

    dropHandler(e) {
        if (e.dataTransfer.items && e.dataTransfer.items.length > 0) {
            let item = e.dataTransfer.items[0];
            if (item.kind === 'file')
                this.video = item.getAsFile();
        }
    }

    submit(e) {
        e.preventDefault();

        if (this.uploadForm.valid && this.video) {
            var fd = new FormData();
            fd.append('video', this.video, this.video.name);
            fd.append('title', this.uploadForm.get('title').value);
            fd.append('description', this.uploadForm.get('description').value);
            fd.append('userID', this.auth.currentUserValue.id?.toString());
            fd.append('categoryIds', '[' + this.uploadForm.get('categories').value + ']');

            this.vidService.upload(fd)
                .subscribe((resp) => {
                    this.router.navigate([this.vidService.BASE_URL, 'v', resp.id])
                });
        }
    }
}
