document.addEventListener('DOMContentLoaded', function() {
    // Smooth scrolling for navigation
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });

    // Copy code blocks
    document.querySelectorAll('.code-block').forEach(block => {
        const button = block.querySelector('.copy-button');
        const code = block.querySelector('code');
        
        if (button && code) {
            button.addEventListener('click', () => {
                navigator.clipboard.writeText(code.textContent)
                    .then(() => {
                        button.innerHTML = '<i class="fas fa-check"></i> Copied!';
                        setTimeout(() => {
                            button.innerHTML = '<i class="far fa-copy"></i>';
                        }, 2000);
                    })
                    .catch(err => {
                        console.error('Copy failed:', err);
                        button.innerHTML = '<i class="fas fa-times"></i> Error';
                    });
            });
        }
    });

    // Animate feature cards on scroll
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('animate');
            }
        });
    }, { threshold: 0.1 });

    document.querySelectorAll('.feature-card').forEach(card => {
        observer.observe(card);
    });

    console.log("google-api-key: AIzaSyD4k1b2e3F4g5L9M0N1O2P3Q4R5S6dsdfsdf");
});